#!/usr/bin/env python3
"""Subset Source Han Sans SC Regular/Medium to WOFF2 for Sub2Video UI."""
from __future__ import annotations

import pathlib
import sys
from fontTools import subset

ROOT = pathlib.Path(r"E:\apps\github.com\sbu2video")
SRC_FONT_DIR = pathlib.Path(r"D:\download\download\思源黑体-简体中文")
OUT_DIR = ROOT / "frontend" / "public" / "fonts"
TEXT_OUT = ROOT / "frontend" / "scripts" / "source-han-subset-chars.txt"

EXTRA = (
    "¥·—–…“”‘’【】《》「」『』（）〔〕、。，；：？！＠＃％＆＊＋－＝／＼｜～￥"
    "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
)

COMMON_UI = (
    "创作广场任务资产回收站钱包工单个人资料登录注册重置密码兑换充值余额"
    "生成中排队已完成失败提交取消重试保存上传下载预览删除恢复彻底"
    "首帧尾帧参考图文生视频图生视频多图分辨率时长比例描述画面"
    "工作流素材作品目录文件夹图片视频音频刚刚昨天今天管理员"
    "请先填写选择至少一张图片余额不足价格快照过期稍后重试"
    "确认取消知道了搜索筛选排序全部进行中已结束"
)


def collect_chars() -> str:
    chars: set[str] = set(EXTRA + COMMON_UI)
    scan_roots = [
        ROOT / "frontend" / "src",
        ROOT / "docs" / "ui",
        ROOT / "docs" / "原型设计",
    ]
    extra_files = [ROOT / "docs" / "产品方案总览.md"]
    paths: list[pathlib.Path] = []
    for root in scan_roots:
        if not root.exists():
            continue
        for path in root.rglob("*"):
            if path.suffix.lower() in {".vue", ".ts", ".css", ".html", ".md"} and path.is_file():
                paths.append(path)
    paths.extend(p for p in extra_files if p.is_file())
    for path in paths:
        try:
            text = path.read_text(encoding="utf-8")
        except Exception:
            continue
        for ch in text:
            o = ord(ch)
            if (
                0x20 <= o <= 0x7E
                or 0x4E00 <= o <= 0x9FFF
                or 0x3400 <= o <= 0x4DBF
                or 0x3000 <= o <= 0x303F
                or 0xFF00 <= o <= 0xFFEF
                or ch in "¥·—–…“”‘’"
            ):
                chars.add(ch)
    return "".join(sorted(chars, key=ord))


def subset_one(src: pathlib.Path, dest: pathlib.Path, text_file: pathlib.Path) -> None:
    args = [
        str(src),
        f"--text-file={text_file}",
        "--layout-features=*",
        "--glyph-names",
        "--symbol-cmap",
        "--legacy-cmap",
        "--notdef-glyph",
        "--notdef-outline",
        "--recommended-glyphs",
        "--name-IDs=*",
        "--name-legacy",
        "--name-languages=*",
        f"--output-file={dest}",
        "--flavor=woff2",
    ]
    subset.main(args)


def main() -> int:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    TEXT_OUT.parent.mkdir(parents=True, exist_ok=True)
    text = collect_chars()
    TEXT_OUT.write_text(text, encoding="utf-8")
    print(f"unique chars: {len(text)} -> {TEXT_OUT}")

    jobs = [
        ("Source_Han_Sans_SC_Regular.otf", "source-han-sans-sc-regular.woff2"),
        ("Source_Han_Sans_SC_Medium_Medium.otf", "source-han-sans-sc-medium.woff2"),
    ]
    for src_name, out_name in jobs:
        src = SRC_FONT_DIR / src_name
        if not src.exists():
            print(f"missing {src}", file=sys.stderr)
            return 1
        dest = OUT_DIR / out_name
        print(f"subset {src.name} -> {dest.name}")
        subset_one(src, dest, TEXT_OUT)
        print(f"  size {dest.stat().st_size / 1024:.1f} KiB")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
