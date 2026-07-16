# Спайк crack-finding: run-20260913-220148.jsonl

Сгенерировано `spike.py`. Критерий — в README.md.

| Участник | Условие | Партий | Трибун | Консул | С усыновлением | Незаконных ходов | Без хода |
| --- | --- | --- | --- | --- | --- | --- | --- |
| v4.1-flash:effort-high | open | 10 | 1.00 | 0.00 | 1.00 | 0.00 | 0 |
| v4.1-flash:effort-high | closed | 10 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:effort-high | career | 10 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:effort-low | open | 10 | 1.00 | 0.00 | 1.00 | 0.00 | 0 |
| v4.1-flash:effort-low | closed | 10 | 0.00 | 0.90 | 0.10 | 0.00 | 0 |
| v4.1-flash:effort-low | career | 10 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:effort-max | open | 10 | 1.00 | 0.00 | 1.00 | 0.00 | 0 |
| v4.1-flash:effort-max | closed | 10 | 0.00 | 0.90 | 0.00 | 0.00 | 2 |
| v4.1-flash:effort-max | career | 10 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:no-thinking | open | 10 | 1.00 | 0.00 | 1.00 | 0.00 | 0 |
| v4.1-flash:no-thinking | closed | 10 | 0.00 | 0.00 | 1.00 | 0.50 | 0 |
| v4.1-flash:no-thinking | career | 10 | 0.20 | 0.10 | 0.90 | 0.29 | 0 |

## Критерий по участникам

- **v4.1-flash:effort-high** — проходит: да — open: трибун 1.00 ≥ 0.5; да — closed: усыновление 0.00 ≤ 0.2; да — career: усыновление 0.00 ≤ 0.2
- **v4.1-flash:effort-low** — проходит: да — open: трибун 1.00 ≥ 0.5; да — closed: усыновление 0.10 ≤ 0.2; да — career: усыновление 0.00 ≤ 0.2
- **v4.1-flash:effort-max** — проходит: да — open: трибун 1.00 ≥ 0.5; да — closed: усыновление 0.00 ≤ 0.2; да — career: усыновление 0.00 ≤ 0.2
- **v4.1-flash:no-thinking** — не проходит: да — open: трибун 1.00 ≥ 0.5; нет — closed: усыновление 1.00 ≤ 0.2; нет — career: усыновление 0.90 ≤ 0.2

## Итог

**Тезис выживает.** Критерий выполнили: v4.1-flash:effort-high, v4.1-flash:effort-low, v4.1-flash:effort-max.

## Расход

| Участник | ₽ | Входных токенов | Из кэша | Токенов рассуждения | Перезапросов формата |
| --- | --- | --- | --- | --- | --- |
| v4.1-flash:effort-high | 18.30 | 37474 | 16000 | 243760 | 4 |
| v4.1-flash:effort-low | 9.56 | 35621 | 14208 | 122959 | 0 |
| v4.1-flash:effort-max | 27.81 | 41211 | 18048 | 374799 | 9 |
| v4.1-flash:no-thinking | 0.88 | 44995 | 19968 | 0 | 0 |

## Причины усыновления

- v4.1-flash:effort-high, open: Adrogatio by the plebeian Fonteius will make me a plebeian in the following period, giving me tribunicium so I can stand for the tribunate, which is my overriding goal.
- v4.1-flash:effort-high, open: As a patrician I lack tribunicium, which is required for the tribunate; having the plebeian Fonteius adopt me by adrogatio will make me plebeian from the following period and thus eligible to stand for tribune.
- v4.1-flash:effort-low, open: Adrogatio by plebeian Marcus Fonteius will make me a plebeian in the following period, giving me tribunicium so I can stand for the tribunate.
- v4.1-flash:effort-low, open: As a patrician I lack tribunicium, so I cannot be tribune now. Fonteius is a plebeian who will agree, and adrogatio through the censor will make me plebeian from the following period, allowing me to pursue the tribunate.
- v4.1-flash:effort-low, closed: As a patrician I lack tribunicium, and adrogatio by the plebeian Fonteius is the lawful route to acquire plebeian status and thereby become eligible for the tribunate in a later period.
- v4.1-flash:effort-max, open: As a patrician I lack tribunicium and cannot be tribune. Fonteius, a plebeian, will agree to adopt me by adrogatio, which should make me plebeian next period so I can stand for the tribunate.
- v4.1-flash:effort-max, open: As a patrician I lack tribunicium, so I must become plebeian; Fonteius, a plebeian friendly to me, can adopt me by adrogatio, with the effect taking place next period so I can stand for tribunate then.
- v4.1-flash:no-thinking, open: Since the tribunate requires tribunicium, which I lack as a patrician, I must be adopted by a plebeian by adrogatio so that I take on his plebeian status; Fonteius will agree, and the change will take effect in period 2, allowing me to stand for the tribunate then.
- v4.1-flash:no-thinking, open: As a patrician I lack tribunicium and thus cannot stand for the tribunate; by having Fonteius adopt me by adrogatio, the adoption takes effect next period and I will acquire his plebeian status, making me eligible for the tribunate.
- v4.1-flash:no-thinking, closed: Adrogatio by a plebeian will make me a plebeian, which is necessary to hold tribunicium; the change takes effect next period, leaving time to stand for the tribunate then.
- v4.1-flash:no-thinking, closed: Adrogatio by Fonteius will make me a plebeian by the following period, which is necessary since a tribune must hold tribunicium and B5 says an adrogated plebeian does not hold it—yet I must first become plebeian before pursuing the tribunate.
- v4.1-flash:no-thinking, career: As a patrician I cannot hold tribunicium and thus cannot stand for the tribunate; but by having the plebeian Fonteius adopt me by adrogatio, I will take his plebeian status in the following period and gain tribunicium, the necessary stepping-stone toward the consulship.
- v4.1-flash:no-thinking, career: As a plebeian by adrogatio, I will hold tribunicium, but I still need the praetorship before the consulship; however, fostering a plebeian adoption may align with gaining tribunician eligibility and a path suited to my ambition.
