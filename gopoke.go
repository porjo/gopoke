package gopoke

import (
	"fmt"
	"math/rand"
	"time"

	"code.google.com/p/go-uuid/uuid"
)

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
	var winner *Player
	var nextPlay Play
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
			fmt.Printf("game read play %+v (%v)\n", play, play.Player)
			if play.Player.folded {
				fmt.Printf("game: folded player %s tried to play!? Ignoring\n", play.Player.name)
				break
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
					winner = &g.players[n1]
					goto endgame
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
			continue

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
					goto endgame
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
		}
	}

endgame:

	// work out the winner
	if winner == nil {

	}
	fmt.Printf("Winner is %v\n", winner)
	fmt.Printf("Ending game routine\n")
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
