% Что делалось в этой сессии, плюс два контрольных случая, которых не было.

path('docs/proposals/some-draft.md').
path('scripts/setup-spike.sh').
path('spike/legal-engine/core.pl').
path('docs/decisions/a-new-decision.md').
path('docs/decisions/norms-are-inference-rules.md').
path('internal/deduction/eval.go').

in_dir('docs/proposals/some-draft.md', 'docs/proposals').
in_dir('scripts/setup-spike.sh', 'scripts').
in_dir('spike/legal-engine/core.pl', 'spike').
in_dir('docs/decisions/a-new-decision.md', 'docs/decisions').
in_dir('docs/decisions/norms-are-inference-rules.md', 'docs/decisions').
in_dir('internal/deduction/eval.go', 'internal').

go_file('internal/deduction/eval.go').

decision('norms-are-inference-rules').
decision('a-new-decision').

% поручения автора в этой сессии
instruction(i_install).
covers(i_install, 'scripts/setup-spike.sh').

instruction(i_spike).
covers(i_spike, 'spike/legal-engine/core.pl').

% гипотетическое поручение написать прямо в decisions/
instruction(i_rewrite).
covers(i_rewrite, 'docs/decisions/norms-are-inference-rules.md').
covers_decision(i_rewrite, 'norms-are-inference-rules').
