#!/usr/bin/env python3
"""Generate the demo GIF used by the README (terminal style); every frame comes from **real command output**.

Usage:
    python3 scripts/make-demo-gif.py [output path, default assets/demo.gif]

Design:
  - canvas = a dark terminal window (title bar + rounded corners), rendered with a monospace font;
  - each command is "typed" character by character, then the real output appears line by line;
  - frames are rendered with PIL -> encoded by ffmpeg with a shared palette (keeps the size in check).

Dependencies: python3-Pillow, ffmpeg, any monospace font (Noto Sans Mono by default).
After regenerating, check the result by eye (frame size 1000x560, size < 1.5 MB).
"""

import os
import shutil
import subprocess
import sys
import tempfile

from PIL import Image, ImageDraw, ImageFont

W, H = 1000, 560
BAR_H = 34
PAD = 22
LINE_H = 26
FONT_CANDIDATES = [
    "/usr/share/fonts/google-noto/NotoSansMono-Regular.ttf",
    "/usr/share/fonts/google-noto/NotoSansMono-SemiCondensedLight.ttf",
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    "/usr/local/share/fonts/m/MapleMono_NF_Base_Mono.ttf",
]
BG = (13, 17, 23)
BAR = (22, 27, 34)
DOT = [(255, 95, 86), (255, 189, 46), (39, 201, 63)]
FG = (201, 209, 217)
DIM = (139, 148, 158)
GREEN = (63, 185, 80)
CYAN = (88, 166, 255)
AMBER = (210, 153, 34)
RED = (248, 81, 73)
BOLD = (240, 246, 252)


def load_font(size):
    for p in FONT_CANDIDATES:
        if os.path.exists(p):
            return ImageFont.truetype(p, size)
    raise SystemExit("no monospace font found; please install Noto Sans Mono or DejaVu Sans Mono")


F = load_font(19)
FB = load_font(20)


def base_frame(title="quarklang — ~/QuarkLang"):
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    d.rounded_rectangle([0, 0, W - 1, BAR_H], radius=0, fill=BAR)
    for i, c in enumerate(DOT):
        x = 18 + i * 20
        d.ellipse([x, BAR_H // 2 - 6, x + 12, BAR_H // 2 + 6], fill=c)
    tw = d.textlength(title, font=F)
    d.text(((W - tw) / 2, 8), title, font=F, fill=DIM)
    return img


class Term:
    """Terminal screen accumulated line by line; every step produces one frame."""

    def __init__(self):
        self.lines = []  # (text, color)
        self.frames = []
        self.title = "quarklang — ~/QuarkLang"

    def _render(self):
        img = base_frame(self.title)
        d = ImageDraw.Draw(img)
        y = BAR_H + PAD
        for text, color in self.lines[-18:]:
            d.text((PAD, y), text, font=F, fill=color)
            y += LINE_H
        return img

    def snap(self, times=1):
        img = self._render()
        for _ in range(times):
            self.frames.append(img.copy())

    def type_cmd(self, cmd, prompt_color=GREEN, step=3):
        """Type character by character (step characters per frame)."""
        for i in range(0, len(cmd) + 1, step):
            self.lines.append(("$ " + cmd[:i], BOLD))
            self.snap()
            self.lines.pop()
        self.lines.append(("$ " + cmd, BOLD))
        self.snap(3)

    def out(self, text, color=FG, pause=2):
        self.lines.append(("  " + text if text else "", color))
        self.snap(pause)

    def out_lines(self, lines, color=FG, pause=1):
        for ln in lines:
            self.out(ln, color, pause)

    def save(self, path):
        fd, tmp = tempfile.mkstemp(suffix=".gif")
        os.close(fd)
        # Write single frames as PNG first, then let ffmpeg build a shared palette (small size, stable colors)
        pngdir = tempfile.mkdtemp()
        for i, fr in enumerate(self.frames):
            fr.save(os.path.join(pngdir, f"f{i:04d}.png"))
        palette = os.path.join(pngdir, "pal.png")
        subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-i",
                        os.path.join(pngdir, "f%04d.png"), "-vf",
                        "palettegen=max_colors=64", palette], check=True)
        subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-framerate", "8",
                        "-i", os.path.join(pngdir, "f%04d.png"), "-i", palette,
                        "-lavfi", "paletteuse=dither=bayer", "-loop", "0", tmp], check=True)
        shutil.move(tmp, path)
        shutil.rmtree(pngdir, ignore_errors=True)
        size = os.path.getsize(path) / 1024
        print(f"✓ wrote {path}: {len(self.frames)} frames, {size:.0f} KB")


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "assets/demo.gif"
    os.makedirs(os.path.dirname(out) or ".", exist_ok=True)
    t = Term()
    t.snap(8)

    # 1) a first look at the language: hello.qk
    t.type_cmd("cat hello.qk")
    t.out_lines([
        'fn main(IOStream io) {',
        '    io.println("Hello World!");',
        '}',
    ], CYAN, pause=6)

    # 2) run it straight through the interpreter
    t.type_cmd("quark hello.qk")
    t.out("Hello World!", FG, pause=6)

    # 3) compile it and run (qkc -run)
    t.type_cmd("qkc -run hello.qk")
    t.out("Hello World!", GREEN, pause=6)

    # 4) static check (real output; English mode, used by the English README)
    t.type_cmd("QK_LANG=en qkcheck examples/")
    t.out("examples/trycatch.qk:3:13: warning: variable a is declared but never used [QK101]", AMBER, pause=6)
    t.out("qkcheck: 1 issue(s) (0 error(s) / 1 warning(s))", DIM, pause=16)
    t.save(out)


if __name__ == "__main__":
    main()
