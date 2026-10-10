# QuarkLang — Interface Inventory and Composition Map

English | [中文](ARCHITECTURE.zh-CN.md)

Status: skeleton (pushed early on purpose; every claim below is replaced with a
`file:line`-referenced claim before this branch is proposed for merge).

This document describes QuarkLang as a set of **interfaces** and the
**compositions** that consume them. It never says "project X is red"; it says
"interface I promises P, and under composition C that promise does not hold".

Sections: 1 interface inventory · 2 composition map · 3 boundaries and
propagation · 4 invariant and ratchet catalogue · 5 worked example ·
6 known gaps.
