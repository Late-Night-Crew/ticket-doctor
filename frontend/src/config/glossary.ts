export interface GlossaryEntry {
  id: string
  term: string
  body: string
}

export const GLOSSARY: GlossaryEntry[] = [
  { id: 'ticket', term: 'Ticket', body: 'Several picks combined into one bet, also called an accumulator. Every pick must win for the ticket to pay out.' },
  { id: 'leg', term: 'Leg', body: 'One pick on a ticket, for example "Arsenal to win".' },
  { id: 'win', term: 'Win (home or away)', body: 'The team you pick must win the match after 90 minutes plus stoppage time. A draw loses.' },
  { id: 'draw', term: 'Draw', body: 'The match must end level after 90 minutes plus stoppage time.' },
  { id: 'double-chance', term: 'Double chance', body: 'Covers two of the three results. 1X means the home team wins or draws; X2 means the away team wins or draws.' },
  { id: 'btts', term: 'BTTS (both teams to score)', body: '"Yes" wins if each team scores at least one goal. "No" wins if at least one team scores none.' },
  { id: 'over-under', term: 'Over / under goals', body: 'Counts total goals by both teams. Over 2.5 needs 3 or more; under 2.5 needs 2 or fewer. The .5 means there is no tie.' },
  { id: 'team-to-score', term: 'Team to score', body: 'The team you pick must score at least one goal. The result does not matter.' },
  { id: 'odds', term: 'Decimal odds', body: 'What a bookmaker pays per 1 staked, including your stake. Odds of 2.50 return 2.50 for every 1 staked if the pick wins.' },
  { id: 'implied', term: 'Implied probability', body: '1 ÷ decimal odds. Odds of 2.00 imply a 50% chance. Bookmakers add a margin, so implied chances are usually a little high.' },
  { id: 'fair-odds', term: 'Fair odds', body: "1 ÷ the app's estimated chance. A 50% estimate gives fair odds of 2.00. This is our estimate only; bookmaker odds will differ." },
  { id: 'combined', term: 'Combined chance', body: 'The chance that every leg wins, found by multiplying the chances of each leg. It falls quickly as you add legs.' },
  { id: 'risk-band', term: 'Risk band', body: 'Low: estimated chance of 65% or more. Medium: 45% to under 65%. High: under 45%. Limited data moves a leg up one band.' },
  { id: 'form-window', term: 'Form window', body: "How many recent home or away matches a team's stats use. The default is the last 10." },
]

const BY_ID = new Map(GLOSSARY.map((g) => [g.id, g]))

export function glossaryEntry(id: string): GlossaryEntry | undefined {
  return BY_ID.get(id)
}
