import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
if str(ROOT) not in sys.path:
    sys.path.append(str(ROOT))

from tools.ocr_parser import parser as ocr_parser

DATA_DIR = Path(__file__).parent / "data"


def load_sample(name: str) -> str:
    return (DATA_DIR / name).read_text(encoding="utf-8")


def test_rechnung_re0039_discount_rows():
    text = load_sample("rechnung_re0039.txt")
    parser = ocr_parser.OCRParser(text)
    items = parser.parse()

    assert len(items) >= 5

    line6 = next(item for item in items if item.line_number == 6)
    assert line6.quantity == 1
    assert round(line6.unit_price, 2) == 250.00
    assert round(line6.discount_percent, 2) == 100.0
    assert round(line6.line_total, 2) == 0.0

    line8 = next(item for item in items if item.line_number == 8)
    assert line8.quantity == 4
    assert round(line8.unit_price, 2) == 20.0
    assert round(line8.discount_percent, 2) == 20.0
    assert round(line8.line_total, 2) == 64.0


def test_extracts_document_title_and_number_from_offer_heading():
    parser = ocr_parser.OCRParser(
        """Tsunami Events UG
Angebot Luther Theater AG0081
Gerne bieten wir Ihnen an:
Pos.
Bezeichnung
Menge
Einheit
Einzel €
Gesamt €
"""
    )

    header = parser.parse_document_header(parser.preprocess())

    assert header == {
        "type": "offer",
        "title": "Luther Theater",
        "number": "AG0081",
    }


def test_keeps_numbered_detail_line_and_repairs_hyphenation():
    parser = ocr_parser.OCRParser(
        """Pos.
Bezeichnung
Menge
Einheit
Einzel €
Gesamt €
5
DPA d:fine CORE 4188 slim
2 Ohr Headset Beige, Niere, 100mm Boom, mi
-
cro dot
7
Stück
40,00
280,00
Gesamtbetrag
280,00
"""
    )

    items = parser.parse()

    assert len(items) == 1
    assert items[0].line_number == 5
    assert items[0].description == (
        "DPA d:fine CORE 4188 slim 2 Ohr Headset Beige, Niere, "
        "100mm Boom, micro dot"
    )
    assert items[0].quantity == 7
    assert items[0].unit_price == 40
    assert items[0].line_total == 280


def test_ignores_page_footer_and_summary_amounts_after_items():
    parser = ocr_parser.OCRParser(
        """Pos.
Bezeichnung
Menge
Einheit
Einzel €
Gesamt €
24
Botex PSA 321
32A-Stromverteiler
1
Stück
20,00
20,00
Zwischensumme
2.944,00
Steuernummer:
03124680200
Pos.
Bezeichnung
Menge
Einheit
Einzel €
Gesamt €
Übertrag
2.944,00
25
Anfahrt/Abfahrt
30
km
1,20
36,00
Zwischensumme (netto)
2.980,00
Umsatzsteuer 19 %
566,20
Gesamtbetrag
3.546,20
"""
    )

    items = parser.parse()

    assert len(items) == 2
    assert items[0].line_number == 24
    assert items[0].description == "Botex PSA 321 32A-Stromverteiler"
    assert items[0].unit_price == 20
    assert items[0].line_total == 20
    assert items[1].line_number == 25
    assert items[1].description == "Anfahrt/Abfahrt"
    assert items[1].quantity == 30
    assert items[1].unit == "km"
    assert items[1].unit_price == 1.2
    assert items[1].line_total == 36

    totals = parser.parse_totals(parser.preprocess())
    assert totals.subtotal == 2980
    assert totals.total == 3546.2
