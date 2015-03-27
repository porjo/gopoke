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

const Check Action = "check"
const Fold Action = "fold"
const Raise Action = "raise"
const Call Action = "call"
const Allin Action = "All in"

type Game struct {
	PlayerPlay  chan Play
	pot         int
	middlecards []Card
	players     []Player
	cards       []Card
	round       int
}

type Player struct {
	name     string
	cards    []Card
	chips    int
	GamePlay chan Play
	game     *Game
	id       uuid.UUID
}

type Play struct {
	Player       *Player
	Amount       int
	Action       Action
	ValidActions []Action
}

type Round struct {
	highbet int
	pot     int
}

type Card struct {
	Suit  string
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

	// 3 cards into the middle
	for i := 0; i < 3; i++ {
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		g.middlecards = append(g.middlecards, card)
	}

	return g
}

func (g *Game) gameRoutine() {
	var round Round
	// start game
	fmt.Printf("Entering game routine\n")
	fmt.Printf("------ Round %d -------\n", g.round)
	// signal first player
	var next int
	play := Play{}
	play.ValidActions = []Action{Check, Fold}
	fmt.Printf("game notify next player, %s\n", g.players[next].name)
	g.players[next].GamePlay <- play
	for {
		select {
		case play = <-g.PlayerPlay:

			switch play.Action {
			case Check:
				// can't check on non-zero high bet
				if round.highbet != 0 {
					play.Action = Fold
					close(play.Player.GamePlay)
					break
				}

				// do nothing
			case Fold:
				// remove player
				close(play.Player.GamePlay)
				break
			case Raise:

				if play.Amount > round.highbet {
					round.highbet = play.Amount
				} else { // not a raise
					if play.Player.chips >= round.highbet {
						if round.highbet == 0 {
							play.Action = Check
						} else {
							play.Action = Call
						}
						play.Amount = round.highbet
					} else {
						play.Action = Fold
						close(play.Player.GamePlay)
						break
					}
				}
				play.Player.chips -= play.Amount
				if play.Player.chips == 0 {
					play.Action = Allin
				}
				round.pot += play.Amount
			case Call:
				if round.highbet == 0 {
					play.Action = Check
				} else {
					play.Player.chips -= round.highbet
				}
				if play.Player.chips == 0 {
					play.Action = Allin
				}
				round.pot += round.highbet
			case Allin:
			}

			fmt.Printf("game: play %s FROM %s\n", play.Action, play.Player.name)

			// broadcast last action to all other players
			for i, p := range g.players {

				if !uuid.Equal(p.id, play.Player.id) {
					fmt.Printf("game: play %s TO %s\n", play.Action, p.name)
					g.players[i].GamePlay <- play
				}

			}

			next++
			if next == len(g.players) {
				next = 0
				fmt.Printf("r:pot %d, g:pot %d\n", round.pot, g.pot)
				g.pot += round.pot
				round = Round{}
				g.round++
				if g.round > 3 {
					goto done
				}
				fmt.Printf("------ Round %d -------\n", g.round)
			}

			// notify next player
			nextplay := Play{}
			nextplay.ValidActions = []Action{Check, Fold}
			fmt.Printf("game notify next player, %s\n", g.players[next].name)
			g.players[next].GamePlay <- nextplay

		}
	}

done:
	fmt.Printf("Ending game routine\n")
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
	p.GamePlay = make(chan Play)
	p.id = uuid.NewRandom()
	g.players = append(g.players, p)
	return nil
}
