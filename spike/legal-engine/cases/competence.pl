competent(O,K)    :- holds(pos(competent(O,K))).
no_competent(O,K) :- holds(neg(competent(O,K))).
may_grant(O,P,S)  :- holds(pos(may_grant(O,P,S))).

office(praetor). office(censor). office(tribunus).
action_kind(grant_status). action_kind(establish_fact). action_kind(veto).
responsibility(praetor, grant_status).
responsibility(censor, establish_fact).
responsibility(tribunus, veto).

app(r_comp,   pos(competent(Off,Kind))) :- responsibility(Off,Kind).
app(r_nocomp, neg(competent(Off,Kind))) :- office(Off), action_kind(Kind).
app(r_grant,  pos(may_grant(Off,marcus,civis))) :-
        competent(Off, grant_status), entitled(marcus, civis).

app(r_c1, pos(entitled(marcus,civis))).
strength(r_c1,defeasible).
strength(r_comp,defeasible). strength(r_nocomp,defeasible).
strength(r_grant,defeasible).
stronger(r_comp, r_nocomp).

?- may_grant(praetor, marcus, civis).
