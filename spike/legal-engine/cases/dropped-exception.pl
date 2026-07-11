app(r_c1, pos(entitled(marcus,civis))).
app(r_c2, neg(entitled(marcus,civis))) :- offense(marcus,_).
strength(r_c1,defeasible). strength(r_c2,defeasible).
stronger(r_c2,r_c1).
?- entitled(marcus,civis).
