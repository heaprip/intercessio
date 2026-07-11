% Проверки самого корпуса — не «допустимо ли действие», а «не противоречит
% ли написанное написанному». Тот же движок, другой вопрос.

source(r_stale_claim, 'AGENTS.md',
       'update chapter 6 when the active work changes').
source(r_accepted_needs_acceptor, 'AGENTS.md',
       'Accepted by is filled in by the author only').

app(r_stale_claim, pos(contradicts(Doc, D))) :-
    claims_no_unaccepted(Doc), decision_status(D, proposed).

app(r_accepted_needs_acceptor, pos(unsigned_acceptance(D))) :-
    decision_status(D, accepted), no_accepted_by(D).

strength(r_stale_claim, strict).
strength(r_accepted_needs_acceptor, strict).

contradicts(Doc, D)     :- holds(pos(contradicts(Doc, D))).
unsigned_acceptance(D)  :- holds(pos(unsigned_acceptance(D))).
