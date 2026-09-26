#!/usr/bin/env python3
"""生成 README 用的演示 GIF（终端风格），帧内容全部来自**真实命令输出**。

用法：
    python3 scripts/make-demo-gif.py [输出路径，默认 assets/demo.gif]

设计：
  - 画布 = 深色终端窗口（标题栏 + 圆角），等宽字体渲染；
  - 每个命令逐字"打字"，随后逐行出现真实输出；
  - 帧图用 PIL 渲染 → ffmpeg 统一调色板编码（体积可控）。

依赖：python3-Pillow、ffmpeg、任一等宽字体（默认 Noto Sans Mono）。
重新生成后请肉眼检查（帧尺寸 1000×560、体积 < 1.5 MB）。
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
    raise SystemExit("未找到等宽字体，请安装 Noto Sans Mono 或 DejaVu Sans Mono")


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
    """按行累积的终端画面；每步生成一帧。"""

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
        """逐字打字（每帧 step 个字符）。"""
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
        # 单帧先出 PNG，再用 ffmpeg 统一调色板（体积小、颜色稳）
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
        print(f"✓ 生成 {path}：{len(self.frames)} 帧，{size:.0f} KB")


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "assets/demo.gif"
    os.makedirs(os.path.dirname(out) or ".", exist_ok=True)
    t = Term()
    t.snap(8)

    # 1) 看一眼语言：hello.qk
    t.type_cmd("cat hello.qk")
    t.out_lines([
        'fn main(IOStream io) {',
        '    io.println("Hello World!");',
        '}',
    ], CYAN, pause=6)

    # 2) 解释器直接跑
    t.type_cmd("quark hello.qk")
    t.out("Hello World!", FG, pause=6)

    # 3) 编译后跑（qkc -run）
    t.type_cmd("qkc -run hello.qk")
    t.out("Hello World!", GREEN, pause=6)

    # 4) 静态检查（真实输出：examples/ 里一处已复核告警）
    t.type_cmd("qkcheck examples/")
    t.out("examples/trycatch.qk:3:13: warning: 变量 a 声明后从未使用 [QK101]", AMBER, pause=6)
    t.out("qkcheck: 1 个问题（0 错误 / 1 警告）", DIM, pause=16)
    t.save(out)


if __name__ == "__main__":
    main()
