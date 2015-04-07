package gopoke

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"code.google.com/p/go-uuid/uuid"
)

var suitWeight = map[string]int{"Spades": 0, "Clubs": 1, "Diamonds": 2, "Hearts": 3}

var suits = [...]string{"Diamonds", "Spades", "Hearts", "Clubs"}
var suitVals = [...]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

type Action string

const Bet Action = "bet"
const Check Action = "check"
const Fold Action = "fold"
const Raise Action = "raise"
const Call Action = "call"
const Allin Action = "All in"

const StartingChips = 50

type Game struct {
	// game reads player plays from this channel
	PlayerPlay chan Play

	pot         int
	middlecards []Card
	players     []Player
	cards       []Card
	round       int
	winner      *Player
}

type Player struct {
	// player reads game plays from this channel
	GamePlay chan Play

	name   string
	cards  []Card
	chips  int
	game   *Game
	id     uuid.UUID
	folded bool
}

type Play struct {
	// player making the play
	Player *Player
	// amount being bet
	Amount int
	// the action the player wants to take
	Action Action
	// actions that the player may take on their next turn
	ValidActions []Action
}

type round struct {
	highbet int
	pot     int
	// dealer position
	dealerIdx int
	// position holding current high bid
	bidderIdx int
}

type Card struct {
	Suit string
	// face cards have the following values:
	//  - Jack  = 11
	//  - Queen = 12
	//  - King  = 13
	//  - Ace   = 14
	Value int
}

// pair, threes, fours
type commonCard struct {
	card  Card
	count int
}

type byValue []Card
type bySuit []Card

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// get player's name
func (p Player) Name() string {
	return p.name
}

// get player's cards
func (p Player) Cards() []Card {
	return p.cards
}

// get player's chips amount
func (p Player) Chips() int {
	return p.chips
}

// returns whether the player has folded or not
func (p Player) Folded() bool {
	return p.folded
}

func NewGame() *Game {
	var card Card
	g := &Game{}

	for _, suit := range suits {
		for _, val := range suitVals {
			card = Card{Suit: suit, Value: val}
			g.cards = append(g.cards, card)
		}
	}

	var shuffled []Card
	for _, idx := range rand.Perm(len(g.cards)) {
		shuffled = append(shuffled, g.cards[idx])
	}

	g.cards = shuffled

	return g
}

func (g *Game) gameRoutine() {
	var r round
	// start game
	fmt.Printf("Entering game routine\n")
	fmt.Printf("------ Round %d ------- %v\n", g.round, g.middlecards)
	// signal first player
	play := Play{}
	play.ValidActions = []Action{Check, Fold}
	fmt.Printf("game notify next player, %s\n", g.players[1].name)
	g.players[1].GamePlay <- play
	for {
		select {
		case play = <-g.PlayerPlay:
			if end := g.gamePlay(play, r); end {
				goto endgame
			}
		}
	}

endgame:
	// work out the winner
	if g.winner == nil {

		for _, p := range g.players {
			if p.folded {
				continue
			}

			hand := append(p.cards, g.middlecards...)
			sort.Sort(bySuit(hand))
			fmt.Printf("game: %s suit hand %v\n", p.name, hand)
			sort.Sort(byValue(hand))
			fmt.Printf("game: %s straight hand %v\n", p.name, hand)

			if isFlush(hand) {
				fmt.Printf("hand is flush %v\n", hand)
			}
			if isStraight(hand) {
				fmt.Printf("hand is straight %v\n", hand)
			}
			c1, c2 := countKind(hand)
			if c1.count > 1 && c2.count == 0 {
				switch c1.count {
				case 2:
					fmt.Printf("hand is a pair %v\n", c1)
				case 3:
					fmt.Printf("hand is 3 of a kind %v\n", c1)
				case 4:
					fmt.Printf("hand is 4 of a kind %v\n", c1)
				}
			}
			if c1.count > 1 && c2.count > 1 {
				if (c1.count == 2 && c2.count == 3) || (c1.count == 3 && c2.count == 2) {
					fmt.Printf("hand is full house %v, %v\n", c1, c2)
				} else {
					fmt.Printf("hand is 2 pairs %v, %v\n", c1, c2)
				}
			}
		}

	}

	fmt.Printf("Winner is %v\n", g.winner)
	fmt.Printf("Ending game routine\n")
}

func (g *Game) gamePlay(play Play, r round) bool {
	var nextPlay Play
	fmt.Printf("game read play %+v (%v)\n", play, play.Player)
	if play.Player.folded {
		fmt.Printf("game: folded player %s tried to play!? Ignoring\n", play.Player.name)
		return false
	}

	g.adjustPlay(&play, &r)

	if play.Action == Fold {
		n1, err := g.getNextPlayerIdx(play.Player)
		if err != nil {
			panic(err)
		}
		n2, _ := g.getNextPlayerIdx(&g.players[n1])

		if n1 == n2 {
			fmt.Printf("game: %s is the winner by default (others folded)\n", g.players[n1].name)
			g.winner = &g.players[n1]
			return true
		}
	}

	fmt.Printf("game: play %s FROM %s, amount %d\n", play.Action, play.Player.name, play.Amount)

	// broadcast last action to all other players
	for i, p := range g.players {

		if !uuid.Equal(p.id, play.Player.id) {
			fmt.Printf("game: play %s TO %s\n", play.Action, p.name)
			g.players[i].GamePlay <- play
		}

	}

	next, err := g.getNextPlayerIdx(play.Player)
	if err != nil {
		panic(err)
	}

	fmt.Printf("game, next %s bidder %s\n", g.players[next].name, g.players[r.bidderIdx].name)
	if next == r.bidderIdx {
		fmt.Printf("game, bidder's return, endgame %v\n", g.players[next])
		goto endround
	}

	// notify next player
	nextPlay = Play{}

	nextPlay.ValidActions = []Action{Fold, Allin}
	if g.players[next].chips > r.highbet {
		if r.highbet == 0 {
			nextPlay.ValidActions = append(nextPlay.ValidActions, Check)
		} else {
			nextPlay.ValidActions = append(nextPlay.ValidActions, Call, Raise)
		}
	}

	fmt.Printf("game notify next player, %v\n", g.players[next])
	g.players[next].GamePlay <- nextPlay
	return false

endround:
	r.dealerIdx, err = g.getNextPlayerIdx(&g.players[r.dealerIdx])
	if err != nil {
		panic(err)
	}
	fmt.Printf("r:pot %d, g:pot %d\n", r.pot, g.pot)
	g.pot += r.pot
	r = round{}
	r.bidderIdx, err = g.getNextPlayerIdx(&g.players[r.dealerIdx])
	if err != nil {
		panic(err)
	}

	for {
		g.round++
		if g.round > 3 {
			return true
		}

		var card Card
		if g.round == 1 {
			// burn
			_, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
			// the flop
			for i := 0; i < 2; i++ {
				card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
				g.middlecards = append(g.middlecards, card)
			}
		}

		// burn
		_, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		// turn/river
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		g.middlecards = append(g.middlecards, card)
		fmt.Printf("------ Round %d ------- %v\n", g.round, g.middlecards)

		if r.bidderIdx != r.dealerIdx {
			break
		}
		// if all players are All-in, then loop
	}

	// First play of new round
	nextPlay = Play{}

	nextPlay.ValidActions = []Action{Fold, Allin, Check, Bet}
	fmt.Printf("game notify next player, %s\n", g.players[r.bidderIdx].name)
	g.players[r.bidderIdx].GamePlay <- nextPlay

	return false
}

func isFlush(hand []Card) bool {
	flush := true
	sort.Sort(bySuit(hand))
	suit := hand[0].Suit
	for i := 1; i < 5; i++ {
		if hand[i].Suit != suit {
			flush = false
		}
	}

	// check reverse order
	if !flush {
		sort.Sort(sort.Reverse(bySuit(hand)))
		suit := hand[0].Suit
		for i := 1; i < 5; i++ {
			if hand[i].Suit != suit {
				flush = false
			}
		}
	}

	return flush
}

func isStraight(hand []Card) bool {
	sort.Sort(byValue(hand))

	if len(hand) > 5 {
		// remove duplicate values
		newhand := make([]Card, len(hand))
		copy(newhand, hand)
		m := map[int]bool{}
		for _, v := range newhand {
			// if high Ace, then include low ace for the 'wheel'
			if v.Value == 14 {
				newhand = append(newhand, Card{Suit: v.Suit, Value: 0})
			}
			if _, seen := m[v.Value]; !seen {
				newhand[len(m)] = v
				m[v.Value] = true
			}
		}
		newhand = newhand[:len(m)]

		// split cards into 5's then run through isStraight again
		rounds := len(newhand) - 4
		for i := 0; i < rounds; i++ {
			if isStraight(newhand[i : i+5]) {
				return true
			}
		}
	}

	straight := true
	last := hand[0].Value
	for i := 1; i < 5; i++ {
		if last-hand[i].Value != 1 {
			straight = false
		}
		last = hand[i].Value
	}

	return straight
}

func countKind(hand []Card) (c1 commonCard, c2 commonCard) {
	sort.Sort(byValue(hand))

	m := map[int]int{}
	for _, v := range hand {
		m[v.Value]++
	}

	for k, v := range m {
		if (k == c1.card.Value || c1.card.Value == 0) && v > c1.count {
			c1.card.Value = k
			c1.count = v
		} else if v > c2.count {
			c2.card.Value = k
			c2.count = v
		}
	}

	return
}

func (g *Game) adjustPlay(play *Play, r *round) {
	if play.Action == Fold {
		play.Player.folded = true
		return
	}
	if play.Action == Check {
		return
	}

	if play.Amount > play.Player.chips {
		play.Amount = play.Player.chips
	}

	if play.Amount == 0 {
		if play.Action != Fold {
			play.Action = Check
			if r.highbet > 0 {
				play.Action = Fold
				play.Player.folded = true
				return
			}
		}
	} else {
		if play.Amount == play.Player.chips {
			play.Action = Allin
		} else if play.Amount == r.highbet {
			play.Action = Call
			if r.highbet == 0 {
				play.Action = Bet
			} else {
				play.Action = Raise
			}
		}
		if play.Amount >= r.highbet {
			r.highbet = play.Amount
			r.bidderIdx, _ = g.getPlayerIdx(play.Player)
		}
	}
	play.Player.chips -= play.Amount
	r.pot += play.Amount
}

func (g *Game) getPlayerIdx(player *Player) (int, error) {
	for i, p := range g.players {
		if uuid.Equal(p.id, player.id) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("player %s not found", player.name)
}

// get next non-folded player with chips > 0
// otherwise return current player
func (g *Game) getNextPlayerIdx(player *Player) (int, error) {
	var found bool
	var current, next int
	for i, p := range g.players {
		if uuid.Equal(p.id, player.id) {
			current = i
			found = true
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("player %s not found", player.name)

	}

	next = current
	for {
		next = (next + 1) % len(g.players)
		if !g.players[next].folded && g.players[next].chips > 0 {
			break
		}
		if next == current {
			break
		}
	}
	return next, nil
}

func (g *Game) Start() ([]*Player, error) {
	if len(g.players) == 0 {
		return nil, fmt.Errorf("Please create players first")
	}

	var playersOut []*Player
	var card Card
	g.PlayerPlay = make(chan Play, len(g.players))
	// first card
	for i, _ := range g.players {
		g.players[i].cards = make([]Card, 0)
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		g.players[i].cards = append(g.players[i].cards, card)
	}

	// second card
	for i, _ := range g.players {
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		g.players[i].cards = append(g.players[i].cards, card)
		playersOut = append(playersOut, &g.players[i])
	}
	go g.gameRoutine()
	return playersOut, nil
}

func (g *Game) NewPlayer(name string) error {
	p := Player{name: name}
	p.chips = StartingChips
	p.GamePlay = make(chan Play)
	p.id = uuid.NewRandom()
	g.players = append(g.players, p)
	return nil
}

func (a byValue) Len() int { return len(a) }

func (a byValue) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byValue) Less(i, j int) bool {
	return a[i].Value > a[j].Value
}

func (a bySuit) Len() int { return len(a) }

func (a bySuit) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a bySuit) Less(i, j int) bool {
	if suitWeight[a[i].Suit] == suitWeight[a[j].Suit] {
		return a[i].Value > a[j].Value
	} else {
		return suitWeight[a[i].Suit] > suitWeight[a[j].Suit]
	}
}
