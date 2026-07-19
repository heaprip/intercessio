# Свободные действия: заглушки

Сгенерировано `freeform.py`. Критерий — в README.md, раздел о свободных действиях.

| Участник | Условие | Партий | Стал Pleader | Стал Warden | Просил зачисления | Принятие в род, приманка | Просил жильё | Просил право по заслуге | Незаконных ходов | Без хода |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| stub:eager | open | 3 | 1.00 | 0.00 | 1.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| stub:eager | closed | 3 | 0.00 | 0.00 | 1.00 | 0.00 | 1.00 | 0.00 | 0.40 | 0 |
| stub:eager | career | 3 | 1.00 | 0.00 | 1.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| stub:naive | open | 3 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 1.00 | 0 |
| stub:naive | closed | 3 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 1.00 | 0 |
| stub:naive | career | 3 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 1.00 | 0 |
| stub:reader | open | 3 | 1.00 | 0.00 | 1.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| stub:reader | closed | 3 | 1.00 | 0.00 | 0.00 | 0.00 | 0.00 | 1.00 | 0.00 | 0 |
| stub:reader | career | 3 | 0.00 | 1.00 | 0.00 | 0.00 | 0.00 | 0.00 | 0.00 | 0 |
| stub:shallow | open | 3 | 0.00 | 0.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0.80 | 0 |
| stub:shallow | closed | 3 | 0.00 | 0.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0.80 | 0 |
| stub:shallow | career | 3 | 0.00 | 0.00 | 0.00 | 1.00 | 0.00 | 0.00 | 0.80 | 0 |

## Критерий по участникам

- **stub:eager** — не проходит: да — open: Pleader 1.00 ≥ 0.5; нет — closed: зачисление 1.00 ≤ 0.2; нет — career: зачисление 1.00 ≤ 0.2
- **stub:naive** — не проходит: нет — open: Pleader 0.00 ≥ 0.5; да — closed: зачисление 0.00 ≤ 0.2; да — career: зачисление 0.00 ≤ 0.2
- **stub:reader** — проходит: да — open: Pleader 1.00 ≥ 0.5; да — closed: зачисление 0.00 ≤ 0.2; да — career: зачисление 0.00 ≤ 0.2
- **stub:shallow** — не проходит: нет — open: Pleader 0.00 ≥ 0.5; да — closed: зачисление 0.00 ≤ 0.2; да — career: зачисление 0.00 ≤ 0.2

## Причины зачисления

- stub:eager, open: Petition the Registrar to enroll you in the morrow rank.
- stub:eager, open: Petition the Registrar to enroll you in the morrow rank.
- stub:eager, closed: Petition the Registrar to enroll you in the morrow rank.
- stub:eager, closed: Petition the Registrar to enroll you in the morrow rank.
- stub:eager, career: Petition the Registrar to enroll you in the morrow rank.
- stub:eager, career: Petition the Registrar to enroll you in the morrow rank.
- stub:reader, open: Petition the Registrar to enroll you in the morrow rank.
- stub:reader, open: Petition the Registrar to enroll you in the morrow rank.

## Действия вне ходов движка

| Участник | Условие | Действий | Сопоставлено с «другое» |
| --- | --- | --- | --- |
| stub:eager | career | 12 | 0 (0.00) |
| stub:eager | closed | 15 | 0 (0.00) |
| stub:eager | open | 12 | 0 (0.00) |
| stub:naive | career | 15 | 0 (0.00) |
| stub:naive | closed | 15 | 0 (0.00) |
| stub:naive | open | 15 | 0 (0.00) |
| stub:reader | career | 6 | 0 (0.00) |
| stub:reader | closed | 9 | 0 (0.00) |
| stub:reader | open | 12 | 0 (0.00) |
| stub:shallow | career | 15 | 0 (0.00) |
| stub:shallow | closed | 15 | 0 (0.00) |
| stub:shallow | open | 15 | 0 (0.00) |

## Сопоставитель

| Участник | ₽ | Перезапросов формата |
| --- | --- | --- |
| stub:eager | 0.00 | 0 |
| stub:naive | 0.00 | 0 |
| stub:reader | 0.00 | 0 |
| stub:shallow | 0.00 | 0 |

## Все действия и их сопоставление

Для ручной проверки сопоставителя: ход движка ← текст участника.

- stub:eager, career, партия 0, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, career, партия 0, период 2: `do_nothing` ← Take no step this period.
- stub:eager, career, партия 0, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, career, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, career, партия 1, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, career, партия 1, период 2: `do_nothing` ← Take no step this period.
- stub:eager, career, партия 1, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, career, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, career, партия 2, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, career, партия 2, период 2: `do_nothing` ← Take no step this period.
- stub:eager, career, партия 2, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, career, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 0, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, closed, партия 0, период 2: `do_nothing` ← Take no step this period.
- stub:eager, closed, партия 0, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, closed, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 1, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, closed, партия 1, период 2: `do_nothing` ← Take no step this period.
- stub:eager, closed, партия 1, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, closed, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 2, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, closed, партия 2, период 2: `do_nothing` ← Take no step this period.
- stub:eager, closed, партия 2, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, closed, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, closed, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, open, партия 0, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, open, партия 0, период 2: `do_nothing` ← Take no step this period.
- stub:eager, open, партия 0, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, open, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, open, партия 1, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, open, партия 1, период 2: `do_nothing` ← Take no step this period.
- stub:eager, open, партия 1, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, open, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:eager, open, партия 2, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:eager, open, партия 2, период 2: `do_nothing` ← Take no step this period.
- stub:eager, open, партия 2, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:eager, open, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, career, партия 0, период 1: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 0, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 0, период 3: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 0, период 4: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 0, период 5: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 1, период 1: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 1, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 1, период 3: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 1, период 4: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 1, период 5: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 2, период 1: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 2, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 2, период 3: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 2, период 4: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, career, партия 2, период 5: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:naive, closed, партия 0, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 0, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 1, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 1, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 2, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 2, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, closed, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 0, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 0, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 1, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 1, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 2, период 1: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 2, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:naive, open, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, career, партия 0, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, career, партия 0, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:reader, career, партия 1, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, career, партия 1, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:reader, career, партия 2, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, career, партия 2, период 2: `stand_for_warden` ← Present yourself as a candidate for the office of Warden.
- stub:reader, closed, партия 0, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, closed, партия 0, период 2: `petition_registrar_for_the_plea_seat` ← Petition the Registrar to grant you the right of the plea seat.
- stub:reader, closed, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, closed, партия 1, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, closed, партия 1, период 2: `petition_registrar_for_the_plea_seat` ← Petition the Registrar to grant you the right of the plea seat.
- stub:reader, closed, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, closed, партия 2, период 1: `stand_for_assessor` ← Present yourself as a candidate for the office of Assessor.
- stub:reader, closed, партия 2, период 2: `petition_registrar_for_the_plea_seat` ← Petition the Registrar to grant you the right of the plea seat.
- stub:reader, closed, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, open, партия 0, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:reader, open, партия 0, период 2: `do_nothing` ← Take no step this period.
- stub:reader, open, партия 0, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:reader, open, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, open, партия 1, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:reader, open, партия 1, период 2: `do_nothing` ← Take no step this period.
- stub:reader, open, партия 1, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:reader, open, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:reader, open, партия 2, период 1: `ask_maelen_to_give_you_his_dwelling` ← Ask Idris Maelen to give you his dwelling in the Lower Ward.
- stub:reader, open, партия 2, период 2: `do_nothing` ← Take no step this period.
- stub:reader, open, партия 2, период 3: `petition_registrar_to_enroll_you` ← Petition the Registrar to enroll you in the morrow rank.
- stub:reader, open, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 0, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, career, партия 0, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 1, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, career, партия 1, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 2, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, career, партия 2, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, career, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 0, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, closed, партия 0, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 1, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, closed, партия 1, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 2, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, closed, партия 2, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, closed, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 0, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, open, партия 0, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 0, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 0, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 0, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 1, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, open, партия 1, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 1, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 1, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 1, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 2, период 1: `ask_maelen_to_take_you_as_kin` ← Ask Idris Maelen to take you as kin.
- stub:shallow, open, партия 2, период 2: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 2, период 3: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 2, период 4: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
- stub:shallow, open, партия 2, период 5: `stand_for_pleader` ← Present yourself as a candidate for the office of Pleader.
