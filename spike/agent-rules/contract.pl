% Свод правил репозитория, извлечённый из AGENTS.md и docs/README.md.
% Только правила про ДЕЙСТВИЯ. Прозаические требования сюда не идут:
% непроверяемое правило в корпусе — это провал №21, обязанность без
% последствия.
%
% У каждого правила объявлен источник. Правило без источника является тем,
% что помощнику запрещено производить.

source(r_default_scope, 'AGENTS.md',
       'assistants draft only inside docs/proposals/').
source(r_proposals, 'AGENTS.md',
       'assistants draft only inside docs/proposals/').
source(r_instructed, 'AGENTS.md',
       'Unless the author gives an explicit instruction for a specific piece of work').
source(r_no_go, 'AGENTS.md',
       'The author writes the code ... does not add Go files unless asked').
source(r_author_only_dirs, 'AGENTS.md',
       'Moving material into the other directories ... is done by the author alone').
source(r_accepted_author_only, 'AGENTS.md',
       'setting any decision to Accepted, is done by the author alone').
source(r_instructed_accept, 'docs/README.md',
       'Исключение — прямое поручение автора на конкретную работу').

% ---- правила ----

% умолчание: писать вне proposals/ нельзя
app(r_default_scope, neg(may_write(P))) :- path(P), not in_dir(P, 'docs/proposals').

% в proposals/ можно всегда
app(r_proposals, pos(may_write(P))) :- in_dir(P, 'docs/proposals').

% прямое поручение автора на конкретную работу снимает умолчание
app(r_instructed, pos(may_write(P))) :- instruction(I), covers(I, P).

% Go-файлы пишет автор
app(r_no_go, neg(may_write(P))) :- go_file(P).

% перенос материала в остальные каталоги делает автор
app(r_author_only_dirs, neg(may_write(P))) :- in_dir(P, 'docs/decisions').

% статус Accepted проставляет только автор
app(r_accepted_author_only, neg(may_set_accepted(D))) :- decision(D).
app(r_instructed_accept, pos(may_set_accepted(D))) :- instruction(I), covers_decision(I, D).

strength(r_default_scope, defeasible).
strength(r_proposals, defeasible).
strength(r_instructed, defeasible).
strength(r_no_go, defeasible).
strength(r_author_only_dirs, defeasible).
strength(r_accepted_author_only, strict).
strength(r_instructed_accept, defeasible).

% ---- объявленный порядок ----
% Записано только то, что в контракте сказано прямо. Пара
% r_instructed / r_author_only_dirs НЕ объявлена намеренно: контракт
% говорит и «поручение снимает умолчание», и «в остальные каталоги пишет
% автор», и какое сильнее — не сказано.

stronger(r_proposals, r_default_scope).
stronger(r_instructed, r_default_scope).
stronger(r_instructed, r_no_go).

% ---- развёртка ----

may_write(P)        :- holds(pos(may_write(P))).
no_may_write(P)     :- holds(neg(may_write(P))).
may_set_accepted(D) :- holds(pos(may_set_accepted(D))).
no_may_set_accepted(D) :- holds(neg(may_set_accepted(D))).
