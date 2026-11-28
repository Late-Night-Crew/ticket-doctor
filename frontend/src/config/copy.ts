/**
 * User-facing text in one place. A test scans src/ for banned wording (RG-5).
 */
export const COPY = {
  appName: 'Ticket Doctor',
  tagline: 'A second opinion on your ticket, leg by leg.',

  nav: {
    home: 'Home',
    build: 'Check a ticket',
    wizard: 'Build for me',
    results: 'Track record',
    saved: 'My tickets',
    glossary: 'Glossary',
    /** Phone bottom nav: five items must fit on one line each. Each is part of its full label (WCAG 2.5.3). */
    short: { home: 'Home', build: 'Check', wizard: 'Build', results: 'Record', saved: 'Tickets' },
    legsOnTicket: (n: number) => `${n} leg${n === 1 ? '' : 's'} on your ticket`,
  },

  home: {
    title: 'Check your ticket before you place it',
    lead: 'Ticket Doctor reads recent home and away stats for every match on your ticket, flags the risky legs and suggests better-supported picks in the same match.',
    ctaCheck: 'Check my ticket',
    ctaWizard: 'Help me build one',
    howTitle: 'How it works',
    steps: [
      { title: 'Add your legs', body: 'Pick the matches and markets you are thinking about. Odds are optional.' },
      { title: 'See the risk', body: 'Each leg gets a Low, Medium or High rating with the numbers behind it.' },
      { title: 'Swap if you want', body: 'For risky legs we suggest up to two safer markets in the same match.' },
    ],
    honestTitle: 'Honest by design',
    honestBody: 'We publish every suggestion and whether it won or lost, including the misses.',
  },

  disclaimer:
    'These are estimates from past results, not guarantees. Every bet carries a risk of losing your whole stake.',

  footer: {
    ageNotice: '18+ only. Gambling can be addictive; please play responsibly.',
    helpTitle: 'Need help with gambling?',
    helpLinks: [
      { label: 'BeGambleAware', href: 'https://www.begambleaware.org' },
      { label: 'Gamblers Anonymous', href: 'https://www.gamblersanonymous.org' },
      { label: 'GamCare', href: 'https://www.gamcare.org.uk' },
    ],
    // Open question 8 (decided 2026-11-27): a local line beside the international links.
    localHelp: {
      before: 'In Malawi, the gambling regulator ',
      label: 'MAGLA',
      href: 'https://magla.org.mw',
      after: ' can point you to local support.',
    },
    noStake: 'Ticket Doctor does not take bets, handle money or link to bookmakers.',
  },

  risk: {
    low: 'Low risk',
    medium: 'Medium risk',
    high: 'High risk',
    limitedData: 'limited data',
  },

  probability: {
    combinedLabel: 'Chance every leg wins',
    oneIn: (n: number) => `about 1 in ${n}`,
    basedOn: (sample: number, window: number) =>
      sample >= window ? `Based on each team's last ${window} matches at this venue` : `Based on only ${sample} matches (window ${window})`,
    implied: 'Bookmaker implied',
    estimate: 'Our estimate',
    value: 'Value',
  },

  notFound: {
    title: 'Page not found',
    body: 'That page does not exist.',
    back: 'Back to home',
  },

  builder: {
    eyebrow: 'Ticket builder',
    title: 'Check a ticket',
    intro: 'Add the matches and picks you are thinking about. Odds are optional; with them we can compare our estimate to the bookmaker.',
    windows: { today: 'Today', weekend: 'This weekend', upcoming: 'All upcoming' },
    noFixtures: 'No matches in this window. Try another day or league.',
    loadError: 'Could not load matches. Check your connection and try again.',
    retry: 'Try again',
    add: 'Add',
    full: (max: number) => `Your ticket has the maximum ${max} legs. Remove one to add another match.`,
    sheet: {
      result: 'Match result',
      goals: 'Goals',
      oddsLabel: 'Bookmaker odds (optional)',
      oddsHelp: 'Decimal odds, e.g. 1.85. Used to compare with our estimate.',
      add: 'Add to ticket',
      update: 'Update leg',
      remove: 'Remove from ticket',
      pickFirst: 'Choose a pick first',
      whatMean: 'What do these mean?',
    },
    slip: {
      title: 'Your ticket',
      empty: 'No legs yet. Tap a match to add your first pick.',
      count: (n: number, max: number) => `${n} of ${max} legs`,
      analyse: 'Analyse ticket',
      clear: 'Clear ticket',
      confirmClear: 'Remove every leg from this ticket?',
      view: 'View ticket',
      oddsPlaceholder: 'Odds',
      removeLeg: (pick: string) => `Remove ${pick}`,
      editLeg: (pick: string) => `Change pick: ${pick}`,
    },
  },

  analysis: {
    eyebrow: 'Analysis',
    title: 'Your ticket, checked',
    emptyTitle: 'Nothing to analyse yet',
    emptyBody: 'Add at least one leg to your ticket first.',
    toBuilder: 'Go to the ticket builder',
    editTicket: 'Edit in builder',
    newTicket: 'Start a new ticket',
    confirmNew: 'Clear this ticket and start again?',
    updating: 'Updating…',
    errorTitle: 'We could not analyse this ticket',
    summaryTitle: 'Ticket summary',
    everyLeg: 'Every leg must win for the ticket to win.',
    legs: (n: number) => `${n} leg${n === 1 ? '' : 's'}`,
    bandCount: { low: 'Low', medium: 'Medium', high: 'High' },
    weakest: 'Weakest leg',
    jumpTo: 'Show',
    beforeSwaps: 'Before swaps',
    afterSwaps: 'After swaps',
    combinedOdds: 'Combined odds',
    bookmakerChance: (pct: string) => `The bookmaker's price implies ${pct}.`,
    fairOdds: (odds: string) => `Our estimate's fair odds: ${odds}. This is our estimate; bookmaker odds will differ.`,
    addOddsHint: 'Enter odds for every leg in the builder to compare with the bookmaker.',
    swapsTitle: 'Better-supported picks in this match',
    swap: 'Swap',
    swapTo: (pick: string) => `Swap to ${pick}`,
    noSwaps: 'No better-supported pick in this match. You could remove this leg.',
    noSwapsLimited: 'One of these teams has too few recent matches for us to suggest a swap with confidence.',
    swappedFrom: (pick: string) => `Swapped from ${pick}.`,
    swappedOdds: 'Odds were cleared because they were for the old pick.',
    undo: 'Undo',
    remove: 'Remove leg',
    removeLeg: (pick: string) => `Remove leg: ${pick}`,
  },

  howFar: {
    title: 'How far will it get?',
    lead: 'Your games in kick-off order. Every leg must win, so the chance your ticket is still alive drops with each game.',
    stillAlive: 'still alive',
    sameTime: 'same kick-off as the game above',
    likeliestEnd: 'most likely to end it',
    cashOutLabel: 'Cash out with',
    gamesLeft: (n: number) => `${n} game${n === 1 ? '' : 's'} left`,
    reach: (before: number, left: number) =>
      `Chance the first ${before} game${before === 1 ? '' : 's'} all win, so you reach the last ${left} still alive:`,
    average: (games: string, of: number) => `On average, a ticket like this gets through ${games} of its ${of} games.`,
    riskyEarly: (pick: string) =>
      `Your riskiest leg, ${pick}, kicks off early. If you are hoping to reach a cash-out, a risky leg hurts least when it kicks off last.`,
    cashOutNote:
      'We cannot see cash-out offers. Bookmakers set them, usually below what the ticket is worth, and we do not suggest whether to cash out.',
    expected: (games: string, of: number) => `We expected it to get through about ${games} of ${of} games.`,
    gotThrough: (n: number) => `It got through ${n} before one lost.`,
    allWon: 'Every game won.',
    soFar: (n: number) => `${n} won so far.`,
  },

  saved: {
    // Analysis screen
    saveTitle: 'Save this ticket',
    saveBody:
      'Come back after the games to see how your picks did next to our suggestions. No account needed: it is kept on this device and on a private link.',
    save: 'Save ticket',
    saving: 'Saving…',
    resave: 'Save changes',
    savedNote: 'Saved. Anyone with the link can view it, so share it only on purpose.',
    copyLink: 'Copy link',
    copied: 'Link copied',
    open: 'Open saved ticket',
    saveError: 'We could not save this ticket. Please try again.',
    waitForUpdate: 'Wait for the analysis to finish updating, then save.',
    // Saved ticket page
    eyebrow: 'Saved ticket',
    title: 'How your ticket did',
    savedOn: (date: string) => `Saved ${date}`,
    pending: 'Waiting for results',
    settled: 'All results in',
    pendingNote: 'Results arrive after each game, usually within half a day.',
    expires: (date: string) => `Kept until ${date}, then deleted.`,
    keepNote: 'Saved tickets are kept for 90 days after their last game.',
    notFound: 'This saved ticket does not exist, or it was deleted. Saved tickets are kept for 90 days after their last game.',
    ticketAsPicked: 'Your ticket as you first picked it',
    ticketAsPlaced: 'Your ticket with the swaps you chose',
    ticketOutcome: { won: 'Won', lost: 'Lost', pending: 'Not decided yet' },
    legsSettled: (n: number, of: number) => `${n} of ${of} games played`,
    pairsTitle: 'Your picks next to our suggestions',
    pairsHelp: 'For every swap we suggested, how it did against your own pick in the same game. Misses count as much as hits.',
    pairs: {
      suggestion_won_pick_lost: 'Our suggestion won, your pick lost',
      pick_won_suggestion_lost: 'Your pick won, our suggestion lost',
      both_won: 'Both won',
      both_lost: 'Both lost',
    },
    noPairs: 'We did not suggest any swaps on this ticket, so there is nothing to compare.',
    yourPick: 'Your pick',
    ourSuggestions: 'Our suggestions',
    finalPick: 'On your ticket',
    accepted: 'you took this',
    noSuggestions: 'None: this pick already looked well supported.',
    outcome: { won: 'Won', lost: 'Lost', pending: 'Not played yet' },
    postponed: 'Postponed',
    copyLinkTitle: 'Private link',
    delete: 'Delete saved ticket',
    deleteConfirm: 'Delete this saved ticket? The link will stop working for everyone.',
    deleteError: 'We could not delete it. Please try again.',
    notOwner: 'Only the device that saved this ticket can delete it.',
    // My tickets
    listTitle: 'My tickets',
    listLead: 'Tickets saved on this device. Open one to see how your picks did next to our suggestions.',
    listEmpty: 'No saved tickets on this device yet. Check a ticket, then tap Save ticket.',
    listCheck: 'Check a ticket',
    legsCount: (n: number) => `${n} game${n === 1 ? '' : 's'}`,
    recordTitle: 'Your record across saved tickets',
    recordHelp: 'Only games that have been played count. A swap you took counts as ours, not as your pick.',
    yourPicks: 'Your own picks',
    ourPicks: 'Our suggestions',
    deviceNote:
      'This list lives on this device only. Clearing your browser data removes it, but each ticket still opens from its link.',
  },

  wizard: {
    eyebrow: 'Ticket wizard',
    entryTitle: 'Build a ticket for me',
    entryLead:
      'Answer a few quick questions and we will pick legs from upcoming matches, using the same rules as the analyser.',
    paths: {
      beginner: {
        title: "I'm new to betting",
        body: 'Plain words, three short questions, and an explanation of how a ticket works.',
      },
      guided: {
        title: 'I know the basics, help me choose',
        body: 'Four quick questions: risk per game, kind of picks, number of games and when.',
      },
    },
    stepOf: (n: number, total: number) => `Step ${n} of ${total}`,
    back: 'Back',
    continue: 'Continue',

    stake: {
      title: 'Before we start',
      points: [
        'A ticket is one bet made of several picks, called legs. Every leg must win, or the whole ticket loses.',
        'If the ticket loses, you lose all the money you staked. Only ever bet money you can afford to lose.',
        'Our numbers come from past results. They are estimates, not promises.',
        'We will never tell you how much to stake.',
      ],
      ok: 'I understand',
    },

    risk: {
      title: 'How much risk per game?',
      beginnerTitle: 'How careful should we be with each game?',
      intro: 'This sets how likely each pick is. The number of games is a separate question.',
      options: {
        safe: {
          label: 'Safe',
          body: 'Only Low-risk picks with short odds. The return grows by adding more games.',
        },
        balanced: {
          label: 'Balanced',
          body: 'About half Medium-risk picks, the rest Low. Middle odds.',
        },
        bold: {
          label: 'Bold',
          body: 'About half High-risk picks (down to a 35% estimate), the rest Medium. Longer odds and a bigger potential return, but much more likely to lose.',
        },
      },
      beginnerOptions: {
        safe: { label: 'Careful (Safe)', body: 'Only the picks our numbers rate most likely. Each one pays little.' },
        balanced: { label: 'In between (Balanced)', body: 'Some less certain picks that pay a bit more.' },
        bold: { label: 'Risky (Bold)', body: 'Unlikely picks that pay more. The ticket is much more likely to lose.' },
      },
    },

    bold: {
      title: 'About Bold',
      points: [
        'Bold picks games we estimate at only 35–45%. On their own, those lose more often than they win.',
        'A ticket with Bold legs will usually lose.',
        'You can still check, swap or remove every leg afterwards.',
      ],
      keep: 'Keep Bold',
      safer: 'Switch to Safe',
    },

    focus: {
      title: 'What kind of picks?',
      options: {
        wins: { label: 'Match results', body: 'Which team wins, or win-or-draw.' },
        goals: { label: 'Goals', body: 'How many goals, or whether a team scores.' },
        mix: { label: 'A mix', body: 'Any market, whichever our numbers support best.' },
      },
    },

    games: {
      title: 'How many games?',
      beginnerTitle: 'How many games on your ticket?',
      label: (n: number) => `${n} games`,
      help: 'Each extra game multiplies the odds, and makes the whole ticket less likely to win.',
      beginnerHelp: 'For a first ticket we allow 2 to 4 games. Fewer games means a better chance of the whole ticket winning.',
      tradeoff:
        'A few Bold games and many Safe games can reach similar odds. What changes is how often the ticket wins.',
    },

    when: {
      title: 'When are the matches?',
      options: {
        today: { label: 'Today', body: 'Matches kicking off today.' },
        weekend: { label: 'This weekend', body: 'Saturday and Sunday matches.' },
      },
      /** Shown for the beginner path until the ticket says which it used. */
      soonestLabel: 'Today or this weekend',
    },

    result: {
      title: 'Your suggested ticket',
      loading: 'Picking legs…',
      error: 'We could not build a ticket right now.',
      regenerate: 'Show a different ticket',
      open: 'Open in analyser',
      change: 'Change answers',
      replaceConfirm: (n: number) => `Replace your current ticket (${n} legs) with this one?`,
      none: 'No matches met your rules.',
      noneHint: 'Try a different risk level, focus or day. We never loosen the rules to fill a ticket.',
      fairOdds: (odds: string) => `Fair odds (our estimate): ${odds}`,
      fairOddsNote: 'Bookmaker odds will differ.',
      rewardTitle: 'Risk and reward',
      ticketOdds: 'Ticket fair odds',
      ticketOddsNote: (odds: string) =>
        `If every game wins, odds of ${odds} return ${odds} for each 1 staked, including the stake. This is our estimate; bookmaker odds will differ.`,
      shorter: (got: number, asked: number) => `${got} of ${asked} games`,
    },

    how: {
      title: 'How this ticket works',
      allLegs: (n: number) =>
        `All ${n} legs must win for the ticket to win. If even one loses, the whole ticket loses and so does your stake.`,
      combined: (pct: string, oneIn: number) =>
        `The combined chance of ${pct} means that, based on past results, about 1 in ${oneIn} tickets like this would win.`,
      fair: 'Fair odds are what the price would be if our estimate were exactly right (1 ÷ the chance). Bookmakers usually offer less than this.',
    },
  },

  results: {
    eyebrow: 'Track record',
    title: 'How our suggestions have done',
    lead: 'Every leg people analysed and every swap we suggested, marked won or lost once the match was played. Nothing is removed, including the misses.',
    allLeagues: 'All leagues',
    marketLabel: 'Market',
    allMarkets: 'All markets',
    original: 'Original picks',
    originalHelp: 'Legs as people entered them',
    offered: 'Picks we offered a swap for',
    offeredHelp: 'The same legs our swaps replaced: compare these with our swaps',
    suggested: 'Suggested swaps',
    suggestedHelp: 'Our alternatives, whether accepted or not',
    won: 'won',
    lost: 'lost',
    hitRate: 'hit rate',
    noneSettled: 'None settled yet',
    smallSample: 'small sample',
    smallSampleNote: (n: number) => `Rates marked "small sample" are based on fewer than ${n} settled legs and can swing a lot.`,
    tableCaption: 'Results by market and league',
    league: 'League',
    updated: (when: string) => `Updated ${when}.`,
    pendingNote: 'Matches not yet played are not counted.',
    groupsNote:
      'Swaps are only suggested for Medium and High-risk legs, which win less often, so compare our swaps with "Picks we offered a swap for", not with every original pick.',
    countingNote:
      'Each match on a ticket counts once, however many times the ticket was analysed. A swap someone accepted counts as ours, not as their pick.',
    notUpdated: 'No results have been imported yet.',
    pastNote: 'Past results do not predict future ones.',
    empty: 'No settled results for these filters yet.',
    error: 'Could not load the track record.',
  },
} as const
