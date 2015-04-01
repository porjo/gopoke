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

func (p Player) Name() string {
	return p.name
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
	fmt.Printf("------ Round %d -------\n", g.round)
	// signal first player
	play := Play{}
	play.ValidActions = []Action{Check, Fold}
	fmt.Printf("game notify next player, %s\n", g.players[1].name)
	g.players[1].GamePlay <- play
	for {
		select {
		case play = <-g.PlayerPlay:
			fmt.Printf("game read play %+v\n", play)
			if play.Player.folded {
				fmt.Printf("game: folded player %s tried to play!? Ignoring\n", play.Player.name)
				break
			}

			fold := g.adjustPlay(&play, &r)

			if fold {
				count := 0
				for i, p := range g.players {
					if p.folded {
						count++
					}
					if count == len(g.players)-1 {
						// All other players folded, winner by default
						fmt.Printf("game: %s is the winner by default\n", play.Player.name)
						winner = &g.players[i]
						goto endgame
					}
				}
				play.Player.folded = true
			}

			fmt.Printf("game: play %s FROM %s, amount %d\n", play.Action, play.Player.name, play.Amount)

			// broadcast last action to all other players
			for i, p := range g.players {

				if !uuid.Equal(p.id, play.Player.id) {
					fmt.Printf("game: play %s TO %s\n", play.Action, p.name)
					g.players[i].GamePlay <- play
				}

			}

			// find this player
			next, _ := g.getPlayerIdx(play.Player)

			// find next player
			for {
				next = (next + 1) % len(g.players)
				if next == r.bidderIdx {
					// call/check finishes round
					if play.Action == Call || play.Action == Check {
						goto endround
					}
				}
				if !g.players[next].folded {
					break
				}
			}

			// notify next player
			nextPlay = Play{}

			nextPlay.ValidActions = []Action{Fold, Allin}
			if g.players[next].chips >= r.highbet {
				nextPlay.ValidActions = append(nextPlay.ValidActions, Raise)
				if r.highbet == 0 {
					nextPlay.ValidActions = append(nextPlay.ValidActions, Check)
				} else {
					nextPlay.ValidActions = append(nextPlay.ValidActions, Call)
				}
			}

			fmt.Printf("game notify next player, %s\n", g.players[next].name)
			g.players[next].GamePlay <- nextPlay
			continue

		endround:
			newDealer := (r.dealerIdx + 1) % len(g.players)
			fmt.Printf("r:pot %d, g:pot %d\n", r.pot, g.pot)
			g.pot += r.pot
			r = round{}
			r.dealerIdx = newDealer
			r.bidderIdx = r.dealerIdx
			g.round++
			if g.round > 3 {
				goto endgame
			}

			var card Card
			if g.round == 1 {
				// burn
				_, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
				// the flop
				for i := 0; i < 3; i++ {
					card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
					g.middlecards = append(g.middlecards, card)
				}
			}

			// burn
			_, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
			// turn/river
			card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
			g.middlecards = append(g.middlecards, card)
			fmt.Printf("------ Round %d -------\n", g.round)

			// First play of new round
			nextPlay = Play{}

			nextPlay.ValidActions = []Action{Fold, Allin, Check, Bet}
			fmt.Printf("game notify next player, %s\n", g.players[r.dealerIdx].name)
			g.players[r.dealerIdx].GamePlay <- nextPlay
		}
	}

endgame:

	// work out the winner
	if winner == nil {

	}
	fmt.Printf("Winner is %v\n", winner)
	fmt.Printf("Ending game routine\n")
}

func (g *Game) adjustPlay(play *Play, r *round) (fold bool) {
	if play.Action == Fold {
		return true
	}
	if play.Amount > play.Player.chips {
		play.Amount = play.Player.chips
	}

	if play.Amount == 0 {
		if play.Action != Fold {
			play.Action = Check
			if r.highbet > 0 {
				play.Action = Fold
				return true
			}
		}
	} else if play.Amount == play.Player.chips {
		play.Action = Allin

	} else if play.Amount == r.highbet {
		play.Action = Call

	} else if play.Amount > r.highbet {
		if r.highbet == 0 {
			play.Action = Bet
		} else {

			play.Action = Raise
		}
		r.highbet = play.Amount
		r.bidderIdx, _ = g.getPlayerIdx(play.Player)
	}
	play.Player.chips -= play.Amount
	r.pot += play.Amount
	return false
}

func (g *Game) getPlayerIdx(player *Player) (int, error) {
	for i, p := range g.players {
		if uuid.Equal(p.id, player.id) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("player %s not found", player.name)
}

func (g *Game) Start() ([]Player, error) {

	if len(g.players) == 0 {
		return nil, fmt.Errorf("Please create players first")
	}

	var playersOut []Player
	var card Card
	g.PlayerPlay = make(chan Play, len(g.players))
	// first card
	for _, p := range g.players {
		p.cards = make([]Card, 0)
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.cards = append(p.cards, card)
	}

	// second card
	for _, p := range g.players {
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.cards = append(p.cards, card)
		playersOut = append(playersOut, p)
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
