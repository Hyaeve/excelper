from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SequenceItem:
    value: int
    inserted: bool


@dataclass(frozen=True)
class FillInstruction:
    start_row: int
    prefix: str
    suffix: str
    items: list[SequenceItem]


def parse_instruction(text: str) -> FillInstruction:
    normalized = text.replace("\n", " ").replace("；", ";").replace("，", ",")
    parts = [part.strip() for part in normalized.split(";") if part.strip()]
    if len(parts) != 4:
        raise ValueError(
            "指令格式错误，应为：起始行;固定前缀;后缀;51 +58 1-10 +5001-5005"
        )

    start_row_text, prefix, suffix, sequence_text = parts

    try:
        start_row = int(start_row_text)
    except ValueError as exc:
        raise ValueError("起始行必须是整数") from exc

    items = _parse_sequence(sequence_text)
    if not items:
        raise ValueError("序列不能为空")

    return FillInstruction(
        start_row=start_row,
        prefix=prefix,
        suffix=suffix,
        items=items,
    )


def _parse_sequence(sequence_text: str) -> list[SequenceItem]:
    normalized = sequence_text.replace("，", " ").replace("、", " ").replace(",", " ")
    tokens = [token.strip() for token in normalized.split() if token.strip()]
    items: list[SequenceItem] = []

    for token in tokens:
        inserted = token.startswith("+")
        number_text = token[1:].strip() if inserted else token

        if "-" in number_text:
            start_text, end_text = number_text.split("-", 1)
            if not start_text.isdigit() or not end_text.isdigit():
                raise ValueError(f"无法解析序列项: {token}")
            start = int(start_text)
            end = int(end_text)
            if end < start:
                raise ValueError(f"区间结束值不能小于开始值: {token}")
            items.extend(
                SequenceItem(value=value, inserted=inserted)
                for value in range(start, end + 1)
            )
            continue

        if not number_text.isdigit():
            raise ValueError(f"无法解析序列项: {token}")

        items.append(SequenceItem(value=int(number_text), inserted=inserted))

    return items
