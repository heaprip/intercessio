app(r_c1, pos(entitled(marcus,civis))).
app(r_c2, neg(entitled(marcus,civis))) :- offense(marcus,theft).
app(r_c3, pos(entitled(marcus,civis))) :- offense(marcus,theft), merit(marcus,valor).
strength(r_c1,defeasible). strength(r_c2,defeasible). strength(r_c3,defeasible).
stronger(r_c3,r_c2). stronger(r_c2,r_c1).
offense(marcus,theft).
?- no_entitled(marcus,civis).
