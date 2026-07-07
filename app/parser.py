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
            "指令格式错误，应为：起始行;固定前缀;后缀;51,插入58,66,插入71"
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
    tokens = [token.strip() for token in sequence_text.split(",") if token.strip()]
    items: list[SequenceItem] = []

    for token in tokens:
        inserted = False
        number_text = token

        if token.startswith("插入"):
            inserted = True
            number_text = token[2:].strip()

        if not number_text.isdigit():
            raise ValueError(f"无法解析序列项: {token}")

        items.append(SequenceItem(value=int(number_text), inserted=inserted))

    return items
