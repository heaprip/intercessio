compl(pos(X), neg(X)).
compl(neg(X), pos(X)).

% правило вытеснено, если есть применимое противоположное, которое оно не бьёт
overruled(R,L) :- app(R,L), compl(L,K), app(S,K), not stronger(R,S).
blocked(R,L)   :- app(R,L), compl(L,K), app(D,K),
                  strength(D,defeater), not stronger(R,D).

holds(L) :- app(R,L), strength(R,strict).
holds(L) :- app(R,L), strength(R,defeasible),
            not overruled(R,L), not blocked(R,L).

entitled(P,S) :- holds(pos(entitled(P,S))).
no_entitled(P,S) :- holds(neg(entitled(P,S))).
