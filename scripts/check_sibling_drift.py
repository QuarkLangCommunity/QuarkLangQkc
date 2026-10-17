#!/usr/bin/env python3
"""Check the declared role of every sibling repository, and detect mirror drift.

The manifest `sibling-repositories.tsv` records one row per repository:
repository name, path in this tree (or `-`), role and, for `legacy` rows,
a SHA-256 digest of the tracked Go files at the measured commit.

Roles:
  legacy      a superseded standalone implementation; its code is frozen and
              its README must point at the maintained path;
  independent a separate project with no counterpart in `cmd/`;
  mirror      a code copy that must stay byte-identical to `cmd/<tool>`.

The check is read-only. It never edits a sibling repository and never needs
network access; the scheduled workflow supplies the checkouts.
"""

from __future__ import annotations

import argparse
import hashlib
import os
import subprocess
import sys
import tempfile

MANIFEST = os.path.join(os.path.dirname(os.path.abspath(__file__)), "sibling-repositories.tsv")
MAIN_TREE = "https://github.com/QuarkLangCommunity/QuarkLangQkc/tree/main/"
LEGACY_MARK = "Superseded. The maintained implementation"
SKIP_DIRS = {".git", ".github", "bin"}
IGNORED_FILES = {"go.mod", "go.sum", "README.md"}


def git(repo: str, *args: str) -> bytes:
    """Run git in a repository and return its stdout byte-for-byte."""
    return subprocess.run(
        ["git", "-C", repo, *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    ).stdout


def git_code_digest(repo: str) -> str:
    """Hash the tracked Go files of a repository, sorted by path."""
    digest = hashlib.sha256()
    entries = []
    for record in git(repo, "ls-tree", "-r", "-z", "HEAD").split(b"\0"):
        if not record:
            continue
        meta, path = record.split(b"\t", 1)
        kind = meta.split()[1]
        if kind != b"blob":
            continue
        name = path.decode()
        if not name.endswith(".go") or name.startswith("bin/"):
            continue
        entries.append((name, meta.split()[2].decode()))
    for name, blob in sorted(entries):
        digest.update(name.encode() + b"\0")
        digest.update(git(repo, "cat-file", "blob", blob))
        digest.update(b"\0")
    return digest.hexdigest()


def dir_code_digest(root: str) -> str:
    """Hash the Go files under a directory, relative to the directory."""
    paths = []
    for base, dirs, files in os.walk(root):
        dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS)
        for name in files:
            if name.endswith(".go") and name not in IGNORED_FILES:
                full = os.path.join(base, name)
                paths.append(os.path.relpath(full, root).replace(os.sep, "/"))
    digest = hashlib.sha256()
    for rel in sorted(paths):
        digest.update(rel.encode() + b"\0")
        with open(os.path.join(root, rel), "rb") as handle:
            digest.update(handle.read())
        digest.update(b"\0")
    return digest.hexdigest()


def read_manifest() -> list[tuple[str, str, str, str]]:
    """Return the manifest rows, ignoring comments and blank lines."""
    rows = []
    with open(MANIFEST, encoding="utf-8") as handle:
        for number, line in enumerate(handle, 1):
            line = line.rstrip("\n")
            if not line or line.startswith("#"):
                continue
            fields = line.split("\t")
            if len(fields) != 4:
                raise SystemExit(f"manifest line {number}: expected 4 tab-separated fields")
            rows.append((fields[0], fields[1], fields[2], fields[3]))
    return rows


def check_legacy(root: str, repo: str, main_path: str, expected: str) -> list[str]:
    """Return the problems in one legacy repository, or an empty list."""
    checkout = os.path.join(root, repo)
    if not os.path.isdir(os.path.join(checkout, ".git")):
        return [f"{repo}: checkout missing"]
    problems = []
    actual = git_code_digest(checkout)
    if actual != expected:
        problems.append(
            f"{repo}: Go code changed since the measurement (expected {expected[:12]}, got {actual[:12]})"
        )
    readme_path = os.path.join(checkout, "README.md")
    if not os.path.isfile(readme_path):
        problems.append(f"{repo}: README.md is missing")
        return problems
    with open(readme_path, encoding="utf-8") as handle:
        readme = handle.read()
    link = MAIN_TREE + main_path
    if LEGACY_MARK not in readme or link not in readme:
        problems.append(f"{repo}: README does not carry the superseded pointer to {main_path}")
    return problems


def check_independent(main_root: str, repo: str) -> list[str]:
    """Return a problem if an independent repository now collides with `cmd/`."""
    name = repo.removeprefix("QuarkLang").lower()
    candidate = os.path.join(main_root, "cmd", name)
    if os.path.isdir(candidate):
        return [f"{repo}: cmd/{name} now exists; reclassify the sibling pair"]
    return []


def check_mirror(root: str, main_root: str, repo: str, main_path: str) -> list[str]:
    """Return a problem if a declared mirror is not byte-identical to its source."""
    sibling = os.path.join(root, repo)
    source = os.path.join(main_root, main_path)
    if not os.path.isdir(sibling):
        return [f"{repo}: checkout missing"]
    if not os.path.isdir(source):
        return [f"{repo}: source {main_path} is missing"]
    left, right = dir_code_digest(sibling), dir_code_digest(source)
    if left != right:
        return [f"{repo}: mirror drift against {main_path} ({left[:12]} != {right[:12]})"]
    return []


def self_test() -> int:
    """Prove the mirror comparator accepts a copy and rejects a mutation."""
    with tempfile.TemporaryDirectory() as tmp:
        source = os.path.join(tmp, "source")
        mirror = os.path.join(tmp, "mirror")
        os.makedirs(source)
        os.makedirs(mirror)
        body = "package main\n\nfunc main() {}\n"
        for directory in (source, mirror):
            with open(os.path.join(directory, "main.go"), "w", encoding="utf-8") as handle:
                handle.write(body)
        if dir_code_digest(source) != dir_code_digest(mirror):
            print("self-test: identical copy rejected", file=sys.stderr)
            return 1
        print("self-test: identical copy accepted")
        with open(os.path.join(mirror, "main.go"), "a", encoding="utf-8") as handle:
            handle.write("\n// drift\n")
        if dir_code_digest(source) == dir_code_digest(mirror):
            print("self-test: mutated copy accepted", file=sys.stderr)
            return 1
        print("self-test: mutated copy rejected")
    return 0


def main(argv: list[str]) -> int:
    """Run the requested check and return a process exit status."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", help="directory containing the sibling checkouts")
    parser.add_argument("--require-all", action="store_true", help="fail when any checkout is missing")
    parser.add_argument("--self-test", action="store_true", help="exercise the drift comparator")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()
    if not args.root:
        parser.error("--root is required unless --self-test is given")

    main_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    rows = read_manifest()
    problems = []
    counts = {"legacy": 0, "independent": 0, "mirror": 0}
    missing = []
    for repo, main_path, role, digest in rows:
        if role not in counts:
            problems.append(f"{repo}: unknown role {role!r}")
            continue
        counts[role] += 1
        checkout = os.path.join(args.root, repo)
        if role != "independent" and not os.path.isdir(checkout):
            missing.append(repo)
            if args.require_all:
                problems.append(f"{repo}: checkout missing under the supplied root")
            else:
                print(f"skip {repo}: checkout not supplied")
            continue
        if role == "legacy":
            found = check_legacy(args.root, repo, main_path, digest)
            if found:
                problems.extend(found)
            else:
                print(f"ok   {repo}: legacy snapshot and superseded pointer")
        elif role == "independent":
            found = check_independent(main_root, repo)
            if found:
                problems.extend(found)
            else:
                print(f"ok   {repo}: independent, no cmd/ counterpart")
        elif role == "mirror":
            found = check_mirror(args.root, main_root, repo, main_path)
            if found:
                problems.extend(found)
            else:
                print(f"ok   {repo}: mirror identical to {main_path}")

    print(
        f"roles: {counts['legacy']} legacy, {counts['independent']} independent, "
        f"{counts['mirror']} mirror; skipped {len(missing)} missing checkout(s)"
    )
    if problems:
        for problem in problems:
            print(f"FAIL {problem}", file=sys.stderr)
        return 1
    print("sibling drift check: 0 problems")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
