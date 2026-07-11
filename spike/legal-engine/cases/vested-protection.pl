% A4 как defeater: приобретённое не отнимается, пока норма в силе.
no_status(P,S) :- holds(neg(status(P,S))).

person(marcus).
status_kind(civis).

app(r_strip, neg(status(P,S))) :- strips_vested(_N,S), applies_to(_N,P).
app(d_a4,    pos(status(P,S))) :- norm_in_force(a4), person(P), status_kind(S).

strength(r_strip, defeasible).
strength(d_a4, defeater).
stronger(d_a4, r_strip).

strips_vested(n5, civis).
applies_to(n5, marcus).
norm_in_force(a4).

?- no_status(marcus, civis).
