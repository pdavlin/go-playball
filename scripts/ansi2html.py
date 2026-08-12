#!/usr/bin/env python3
"""Convert ANSI (SGR) terminal capture to an HTML fragment.

Reads one or more captures and emits a single HTML page with each frame
in a labeled <pre> block. Usage:
    ansi2html.py out.html "Label A" a.ansi "Label B" b.ansi ...
"""
import html
import re
import sys

CSI = re.compile(r"\x1b\[([0-9;]*)m")

BASE16 = [
    (0, 0, 0), (205, 49, 49), (13, 188, 121), (229, 229, 16),
    (36, 114, 200), (188, 63, 188), (17, 168, 205), (229, 229, 229),
    (102, 102, 102), (241, 76, 76), (35, 209, 139), (245, 245, 67),
    (59, 142, 234), (214, 112, 214), (41, 184, 219), (255, 255, 255),
]


def color256(n):
    if n < 16:
        return BASE16[n]
    if n < 232:
        n -= 16
        r, g, b = n // 36, (n % 36) // 6, n % 6
        conv = lambda v: 0 if v == 0 else 55 + v * 40
        return (conv(r), conv(g), conv(b))
    v = 8 + (n - 232) * 10
    return (v, v, v)


def render(text):
    out = []
    fg = bg = None
    bold = False
    dim = False

    def style():
        f, b = fg, bg
        if f is None and b is None and not bold and not dim:
            return None
        css = []
        if f:
            css.append("color:rgb(%d,%d,%d)" % f)
        if b:
            css.append("background:rgb(%d,%d,%d)" % b)
        if bold:
            css.append("font-weight:bold")
        if dim and not f:
            css.append("opacity:0.55")
        return ";".join(css)

    pos = 0
    open_span = False
    for m in CSI.finditer(text):
        chunk = text[pos:m.start()]
        if chunk:
            out.append(html.escape(chunk))
        pos = m.end()
        params = [int(p) if p else 0 for p in (m.group(1) or "0").split(";")]
        i = 0
        while i < len(params):
            p = params[i]
            if p == 0:
                fg = bg = None
                bold = dim = False
            elif p == 1:
                bold = True
            elif p == 2:
                dim = True
            elif p in (21, 22):
                bold = dim = False
            elif 30 <= p <= 37:
                fg = BASE16[p - 30]
            elif 90 <= p <= 97:
                fg = BASE16[p - 90 + 8]
            elif 40 <= p <= 47:
                bg = BASE16[p - 40]
            elif 100 <= p <= 107:
                bg = BASE16[p - 100 + 8]
            elif p == 39:
                fg = None
            elif p == 49:
                bg = None
            elif p in (38, 48) and i + 1 < len(params):
                target = "fg" if p == 38 else "bg"
                if params[i + 1] == 5 and i + 2 < len(params):
                    c = color256(params[i + 2])
                    i += 2
                elif params[i + 1] == 2 and i + 4 < len(params):
                    c = (params[i + 2], params[i + 3], params[i + 4])
                    i += 4
                else:
                    c = None
                if c is not None:
                    if target == "fg":
                        fg = c
                    else:
                        bg = c
            i += 1
        if open_span:
            out.append("</span>")
            open_span = False
        s = style()
        if s:
            out.append('<span style="%s">' % s)
            open_span = True
    tail = text[pos:]
    if tail:
        out.append(html.escape(tail))
    if open_span:
        out.append("</span>")
    return "".join(out)


def main():
    out_path = sys.argv[1]
    pairs = list(zip(sys.argv[2::2], sys.argv[3::2]))
    blocks = []
    for label, path in pairs:
        with open(path, encoding="utf-8", errors="replace") as f:
            body = render(f.read())
        blocks.append(
            '<h2>%s</h2><pre>%s</pre>' % (html.escape(label), body)
        )
    page = (
        "<!DOCTYPE html><html><head><meta charset='utf-8'>"
        "<style>body{background:#14151a;color:#c8c8c8;font-family:sans-serif;padding:16px}"
        "h2{font-size:14px;color:#9aa;margin:24px 0 6px}"
        "pre{background:#0d0e12;color:#c8c8c8;padding:12px;border-radius:6px;"
        "font-family:'SF Mono',Menlo,monospace;font-size:12px;line-height:1.25;"
        "overflow-x:auto}</style></head><body>%s</body></html>" % "".join(blocks)
    )
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(page)


if __name__ == "__main__":
    main()
