from __future__ import annotations

import argparse

from app.excel_writer import fill_workbook
from app.parser import parse_instruction


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="根据简化语法批量填充 Excel B 列区号"
    )
    parser.add_argument("--input", required=True, help="输入 Excel 文件路径")
    parser.add_argument("--output", required=True, help="输出 Excel 文件路径")
    parser.add_argument(
        "--sheet",
        default="",
        help="工作表名称，不传则使用当前活动工作表",
    )
    parser.add_argument(
        "--instruction",
        required=True,
        help="指令格式：924;25B140-;自交;51 +58 66 +71 73 76",
    )
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    instruction = parse_instruction(args.instruction)
    fill_workbook(
        input_path=args.input,
        output_path=args.output,
        sheet_name=args.sheet,
        instruction=instruction,
    )


if __name__ == "__main__":
    main()
