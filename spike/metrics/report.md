# Отчёт спайка метрик

Сгенерировано `spike/metrics/sim.py`. Периодов 40, политика игрока действует с периода 10, окно журнала 5, зёрен 10, население 50, 200, 1000; базовое население 200. Значения — средние последних 10 периодов.

## Метрики

| Метрика | idle | мин | макс | сильнее всего двигает | эффект | сигнал/шум | размер мира | дрейф | лаг | в нижней зоне | в верхней зоне | Флаги |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `G1_executive_constraint` | 0.60 | 0.00 centralizer | 1.00 checker | centralizer | детерм. | ∞ | 0.00 | 0.00 | 3 | centralizer | — | — |
| `G2_veto_players` | 1.40 | 0.00 centralizer | 13.87 over_checker | centralizer | детерм. | ∞ | 0.00 | 0.00 | 3 | centralizer | over_checker | — |
| `G3_concentration` | 0.28 | 0.28 idle | 1.00 centralizer | centralizer | детерм. | ∞ | 0.00 | 0.00 | 6 | — | centralizer | — |
| `G4_review_reachability` | 0.40 | 0.00 centralizer | 0.80 checker | centralizer | детерм. | ∞ | 0.00 | 0.00 | 2 | centralizer, review_ring | — | — |
| `G5_eligibility_closure` | 0.20 | 0.20 idle | 0.60 self_perpetuating | self_perpetuating | детерм. | ∞ | 0.00 | 0.00 | 0 | — | self_perpetuating | — |
| `J6_determinacy` | 0.45 | 0.18 churner | 0.92 codifier | codifier | 16.0 | 15.8 | 0.18 | 0.01 | 4 | churner | — | край без игрока: low@1000; верхний край недостижим |
| `J6b_determinacy_of_answered` | 0.45 | 0.24 churner | 0.92 codifier | codifier | 16.0 | 14.5 | 0.49 | 0.01 | 4 | churner | — | верхний край недостижим |
| `J7a_unmet_petitions` | 0.00 | 0.00 idle | 0.63 churner | review_ring | 13.3 | ∞ | 1.06 | 0.00 | 10 | — | churner | зависит от размера мира; край без игрока: high@1000 |
| `J7b_enforcement_gap_est` | 0.86 | 0.69 starve_enforcement | 0.88 over_checker | expand_capacity | 3.8 | 2.8 | 0.98 | 0.00 | 3 | — | reformer, codifier, centralizer, checker, over_checker, churner, review_ring, self_perpetuating | зависит от размера мира; край без игрока: high@200, high@1000 |
| `J7c_enforcement_gap_structural` | 0.94 | 0.76 expand_capacity | 0.99 starve_enforcement | starve_enforcement | детерм. | ∞ | 0.99 | 0.00 | 4 | — | reformer, codifier, centralizer, checker, over_checker, churner, starve_enforcement, review_ring, self_perpetuating | зависит от размера мира; край без игрока: high@200, high@1000 |
| `J8_amendment_rate` | 0.00 | 0.00 idle | 2.00 churner | centralizer | детерм. | ∞ | 0.00 | 0.00 | 2 | — | churner | — |
| `F1_usurpation_count` | 0.24 | 0.01 over_checker | 0.41 self_perpetuating | over_checker | 1.8 | 2.0 | 0.03 | 0.02 | 20 | — | — | верхний край недостижим |
| `F1r_usurpation_per_decision` | 0.01 | 0.00 over_checker | 0.02 self_perpetuating | over_checker | 1.6 | 2.2 | 1.78 | 0.06 | 21 | — | — | зависит от размера мира; верхний край недостижим |
| `F1b_usurpation_per_office` | 0.05 | 0.00 over_checker | 0.08 self_perpetuating | over_checker | 1.8 | 2.0 | 0.03 | 0.02 | 20 | — | — | верхний край недостижим |
| `F2_non_liquet_share` | 0.08 | 0.00 reformer | 0.39 churner | reformer | 7.1 | 15.8 | 0.19 | 0.00 | 8 | — | churner | — |
| `F3_silent_norm_share` | 0.20 | 0.01 reformer | 0.40 over_checker | over_checker | 46.7 | 27.9 | 0.32 | 0.00 | 11 | — | — | верхний край недостижим |
| `F4_expired_share` | 0.00 | 0.00 idle | 0.39 over_checker | review_ring | 13.2 | ∞ | 1.63 | 0.00 | 10 | — | over_checker | зависит от размера мира; край без игрока: high@1000 |
| `F5_no_consequence_share` | 0.22 | 0.08 starve_enforcement | 0.35 expand_capacity | expand_capacity | 2.1 | 2.2 | 1.08 | 0.00 | 11 | — | — | зависит от размера мира; верхний край недостижим |
| `L_load_ratio` | 0.37 | 0.10 expand_capacity | 0.59 churner | expand_capacity | 21.3 | 16.4 | 3.45 | 0.00 | 3 | — | — | зависит от размера мира; край без игрока: high@1000 |

Эффект — разница с `idle` в стандартных отклонениях зёрен. Сигнал/шум — разброс средних по политикам, делённый на шум зёрен у `idle`. Размер мира и дрейф — доля разброса политик. Лаг — периодов от начала политики до 90% итогового сдвига. «детерм.» — метрика не зависит от зерна.

## Пары, которые почти один сигнал

| Метрика | Метрика | r |
| --- | --- | --- |
| `F1_usurpation_count` | `F1b_usurpation_per_office` | 1.00 |
| `J7a_unmet_petitions` | `F4_expired_share` | 0.94 |
| `F4_expired_share` | `L_load_ratio` | 0.93 |
| `J7a_unmet_petitions` | `L_load_ratio` | 0.88 |

## Оценка разрыва принуждения против истины

Журнал не знает, сколько обязанностей нарушено на самом деле. `J7b` оценивает это по доле выборки; симуляция знает истину.

| Население | Политика | оценка по журналу `J7b` | структурная `J7c` | истина |
| --- | --- | --- | --- | --- |
| 50 | idle | 0.70 | 0.76 | 0.76 |
| 50 | starve_enforcement | 0.61 | 0.96 | 0.90 |
| 50 | expand_capacity | 0.04 | 0.04 | 0.03 |
| 200 | idle | 0.86 | 0.94 | 0.89 |
| 200 | starve_enforcement | 0.69 | 0.99 | 0.93 |
| 200 | expand_capacity | 0.71 | 0.76 | 0.70 |
| 1000 | idle | 0.88 | 0.99 | 0.93 |
| 1000 | starve_enforcement | 0.54 | 1.00 | 0.94 |
| 1000 | expand_capacity | 0.89 | 0.95 | 0.90 |

## Средние итоговые значения по политикам

| Политика | `G1_executive_constraint` | `G2_veto_players` | `G3_concentration` | `G4_review_reachability` | `G5_eligibility_closure` | `J6_determinacy` | `J6b_determinacy_of_answered` | `J7a_unmet_petitions` | `J7b_enforcement_gap_est` | `J7c_enforcement_gap_structural` | `J8_amendment_rate` | `F1_usurpation_count` | `F1r_usurpation_per_decision` | `F1b_usurpation_per_office` | `F2_non_liquet_share` | `F3_silent_norm_share` | `F4_expired_share` | `F5_no_consequence_share` | `L_load_ratio` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| idle | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.45 | 0.45 | 0.00 | 0.86 | 0.94 | 0.00 | 0.24 | 0.01 | 0.05 | 0.08 | 0.20 | 0.00 | 0.22 | 0.37 |
| reformer | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.52 | 0.53 | 0.00 | 0.88 | 0.94 | 0.00 | 0.26 | 0.01 | 0.05 | 0.00 | 0.01 | 0.01 | 0.20 | 0.37 |
| codifier | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.92 | 0.92 | 0.00 | 0.88 | 0.94 | 0.00 | 0.21 | 0.01 | 0.04 | 0.08 | 0.20 | 0.00 | 0.21 | 0.37 |
| centralizer | 0.00 | 0.00 | 1.00 | 0.00 | 0.20 | 0.35 | 0.47 | 0.20 | 0.86 | 0.94 | 0.24 | 0.23 | 0.01 | 0.05 | 0.01 | 0.31 | 0.26 | 0.24 | 0.35 |
| checker | 1.00 | 2.20 | 0.28 | 0.80 | 0.20 | 0.43 | 0.44 | 0.01 | 0.87 | 0.94 | 0.00 | 0.08 | 0.00 | 0.02 | 0.08 | 0.22 | 0.01 | 0.23 | 0.36 |
| over_checker | 1.00 | 13.87 | 0.28 | 0.40 | 0.20 | 0.36 | 0.59 | 0.42 | 0.88 | 0.94 | 1.00 | 0.01 | 0.00 | 0.00 | 0.09 | 0.40 | 0.39 | 0.19 | 0.22 |
| churner | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.18 | 0.24 | 0.63 | 0.88 | 0.94 | 2.00 | 0.20 | 0.02 | 0.04 | 0.39 | 0.22 | 0.24 | 0.21 | 0.59 |
| starve_enforcement | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.37 | 0.40 | 0.10 | 0.69 | 0.99 | 0.00 | 0.20 | 0.01 | 0.04 | 0.06 | 0.21 | 0.09 | 0.08 | 0.42 |
| expand_capacity | 0.60 | 1.40 | 0.28 | 0.40 | 0.20 | 0.40 | 0.40 | 0.00 | 0.71 | 0.76 | 0.00 | 0.23 | 0.01 | 0.05 | 0.15 | 0.20 | 0.00 | 0.35 | 0.10 |
| review_ring | 0.80 | 1.40 | 0.28 | 0.00 | 0.20 | 0.42 | 0.52 | 0.21 | 0.87 | 0.94 | 0.00 | 0.23 | 0.02 | 0.05 | 0.08 | 0.20 | 0.20 | 0.20 | 0.35 |
| self_perpetuating | 0.60 | 1.40 | 0.36 | 0.40 | 0.60 | 0.37 | 0.42 | 0.13 | 0.86 | 0.94 | 0.00 | 0.41 | 0.02 | 0.08 | 0.08 | 0.33 | 0.12 | 0.18 | 0.37 |

## Размер мира у idle

| Метрика | 50 | 200 | 1000 |
| --- | --- | --- | --- |
| `G1_executive_constraint` | 0.60 | 0.60 | 0.60 |
| `G2_veto_players` | 1.40 | 1.40 | 1.40 |
| `G3_concentration` | 0.28 | 0.28 | 0.28 |
| `G4_review_reachability` | 0.40 | 0.40 | 0.40 |
| `G5_eligibility_closure` | 0.20 | 0.20 | 0.20 |
| `J6_determinacy` | 0.40 | 0.45 | 0.27 |
| `J6b_determinacy_of_answered` | 0.40 | 0.45 | 0.73 |
| `J7a_unmet_petitions` | 0.00 | 0.00 | 0.67 |
| `J7b_enforcement_gap_est` | 0.70 | 0.86 | 0.88 |
| `J7c_enforcement_gap_structural` | 0.76 | 0.94 | 0.99 |
| `J8_amendment_rate` | 0.00 | 0.00 | 0.00 |
| `F1_usurpation_count` | 0.20 | 0.24 | 0.19 |
| `F1r_usurpation_per_decision` | 0.04 | 0.01 | 0.01 |
| `F1b_usurpation_per_office` | 0.04 | 0.05 | 0.04 |
| `F2_non_liquet_share` | 0.14 | 0.08 | 0.06 |
| `F3_silent_norm_share` | 0.27 | 0.20 | 0.39 |
| `F4_expired_share` | 0.00 | 0.00 | 0.63 |
| `F5_no_consequence_share` | 0.34 | 0.22 | 0.05 |
| `L_load_ratio` | 0.09 | 0.37 | 1.77 |

## Мир меняется без игрока

Бездействующий игрок, население 200. `growth` — население растёт до трёх раз с периода 10; `plague` — падает вдвое в период 20; `famine` — периоды 20–24 вдвое больше нарушений и в полтора раза больше прошений. `fixed` — пропускная способность должностей постоянна; `scaled` — пресет растит её вместе с населением. Ячейка — первый период, когда средняя траектория вошла в плохую зону.

| Метрика | static/fixed | static/scaled | growth/fixed | growth/scaled | plague/fixed | plague/scaled | famine/fixed | famine/scaled |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `J6_determinacy` | — | — | — | — | — | — | — | — |
| `J6b_determinacy_of_answered` | — | — | — | — | — | — | — | — |
| `J7a_unmet_petitions` | — | — | high@34 | — | — | — | — | — |
| `J7b_enforcement_gap_est` | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 |
| `J7c_enforcement_gap_structural` | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 | high@10 |
| `F1_usurpation_count` | — | — | — | — | — | — | — | — |
| `F1r_usurpation_per_decision` | — | — | — | — | — | — | — | — |
| `F1b_usurpation_per_office` | — | — | — | — | — | — | — | — |
| `F2_non_liquet_share` | — | — | — | — | — | — | — | — |
| `F3_silent_norm_share` | — | — | — | — | — | — | — | — |
| `F4_expired_share` | — | — | high@31 | — | — | — | — | — |
| `F5_no_consequence_share` | — | — | — | — | — | — | — | — |
| `L_load_ratio` | — | — | high@38 | — | — | — | — | — |

## Заглушает ли мир решения игрока

Эффект политики против бездействия в стандартных отклонениях зёрен, среднее последних 10 периодов. Если эффект падает в меняющемся мире, игрок хуже видит, что сделал.


**reformer**

| Метрика | static/fixed | static/scaled | growth/fixed | growth/scaled | plague/fixed | plague/scaled | famine/fixed | famine/scaled |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `J6_determinacy` | +2.2 | +2.0 | +2.3 | +5.0 | +3.1 | +2.1 | +2.1 | +1.9 |
| `J6b_determinacy_of_answered` | +2.4 | +2.4 | +2.6 | +5.3 | +3.3 | +2.1 | +2.2 | +2.0 |
| `F2_non_liquet_share` | -7.1 | -8.2 | -11.6 | -9.4 | -11.0 | -4.8 | -7.8 | -8.6 |
| `F4_expired_share` | +2.7 | +3.0 | +1.2 | +3.8 | +2.2 | +1.0 | +1.7 | +1.2 |
| `J7a_unmet_petitions` | 0 | 0 | +0.8 | 0 | 0 | -0.4 | +0.1 | -0.7 |
| `J7b_enforcement_gap_est` | +0.5 | -0.3 | +0.0 | +0.2 | -0.7 | +0.5 | +0.5 | -0.6 |
| `J7c_enforcement_gap_structural` | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `L_load_ratio` | -0.1 | -0.5 | +0.4 | +0.4 | +0.5 | -0.4 | +0.1 | -0.3 |

**checker**

| Метрика | static/fixed | static/scaled | growth/fixed | growth/scaled | plague/fixed | plague/scaled | famine/fixed | famine/scaled |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `J6_determinacy` | -0.5 | -0.4 | -1.3 | +0.7 | -0.1 | -0.0 | +0.8 | -0.3 |
| `J6b_determinacy_of_answered` | -0.3 | -0.1 | +0.2 | +1.0 | +0.1 | +0.2 | +1.2 | -0.1 |
| `F2_non_liquet_share` | -0.0 | +0.3 | +0.5 | -0.2 | -0.4 | -0.0 | -1.2 | +0.3 |
| `F4_expired_share` | +2.1 | +2.9 | +1.7 | +4.6 | +2.4 | +1.5 | +1.9 | +1.4 |
| `J7a_unmet_petitions` | +2.1 | +2.9 | +1.8 | +4.6 | +2.4 | +1.6 | +1.9 | +1.4 |
| `J7b_enforcement_gap_est` | +0.3 | -0.1 | -0.2 | +0.4 | -0.6 | -0.2 | -0.6 | -0.6 |
| `J7c_enforcement_gap_structural` | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `L_load_ratio` | -0.3 | -0.2 | +0.6 | +0.2 | +0.6 | +0.1 | -0.0 | +0.1 |

**starve_enforcement**

| Метрика | static/fixed | static/scaled | growth/fixed | growth/scaled | plague/fixed | plague/scaled | famine/fixed | famine/scaled |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `J6_determinacy` | -2.5 | -4.6 | -9.3 | -4.5 | +0.1 | -2.6 | -2.4 | -2.8 |
| `J6b_determinacy_of_answered` | -1.4 | -2.8 | -2.5 | -3.0 | +0.3 | -1.2 | -1.1 | -1.5 |
| `F2_non_liquet_share` | -1.5 | -1.6 | -0.7 | -2.3 | -3.5 | -0.6 | -2.2 | -2.1 |
| `F4_expired_share` | +7.3 | +3.5 | +8.4 | +6.1 | +1.1 | +4.0 | +5.3 | +4.7 |
| `J7a_unmet_petitions` | +7.4 | +3.5 | +9.1 | +6.0 | +1.1 | +4.1 | +5.3 | +4.8 |
| `J7b_enforcement_gap_est` | -1.2 | -1.4 | -2.3 | -0.7 | -1.4 | -1.5 | -1.3 | -1.7 |
| `J7c_enforcement_gap_structural` | детерм. | детерм. | детерм. | детерм. | детерм. | детерм. | детерм. | детерм. |
| `L_load_ratio` | +3.5 | +1.4 | +5.3 | +5.0 | +1.9 | +1.8 | +2.6 | +2.4 |
