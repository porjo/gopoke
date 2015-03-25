package gopoke

import (
	"fmt"
	"math/rand"
	"time"

	"code.google.com/p/go-uuid/uuid"
)

var suits = [...]string{"Diamonds", "Spades", "Hearts", "Clubs"}
var suitVals = [...]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

type Game struct {
	Action      chan Action
	Pot         int
	MiddleCards []Card
	players     []Player
	cards       []Card
	round       int
}

type Action interface {
	GetPlayer() *Player
	String() string
}

type Player struct {
	Name  string
	Cards []Card
	Chips int
	Plays chan Play
	game  *Game
	id    uuid.UUID
}

type Play struct {
	PlayerName   string
	Amount       int
	Action       Action
	ValidActions []Action
}

type Card struct {
	Suit  string
	Value int
}

type Check struct {
	Player *Player
}

func (c Check) GetPlayer() *Player {
	return c.Player
}
func (c Check) String() string {
	return "Check"
}

type Fold struct {
	Player *Player
}

func (c Fold) GetPlayer() *Player {
	return c.Player
}
func (c Fold) String() string {
	return "Fold"
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
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
		g.MiddleCards = append(g.MiddleCards, card)
	}

	return g
}

func (g *Game) gameRoutine() {
	// start game
	fmt.Printf("Entering game routine\n")
	play := Play{}
	play.ValidActions = []Action{Check{}, Fold{}}
	g.players[0].Plays <- play
	for {
		select {
		case a := <-g.Action:

			switch a.(type) {
			case Check:
				// do nothing
			case Fold:
				// remove player
			}

			fmt.Printf("game: action %s FROM %s\n", a, a.GetPlayer().Name)

			var next int
			// broadcast last action to all other players
			for i, p := range g.players {
				if uuid.Equal(p.id, a.GetPlayer().id) {
					next = (i + 1) % len(g.players)
				}
				play = Play{}
				play.Action = a

				if !uuid.Equal(p.id, a.GetPlayer().id) {
					fmt.Printf("game: play %v TO %s\n", play, p.Name)
					g.players[i].Plays <- play
				}

			}

			// notify next player
			play = Play{}
			play.ValidActions = []Action{Check{}, Fold{}}
			fmt.Printf("game notify next player, %s\n", g.players[next].Name)
			g.players[next].Plays <- play

			if next == len(g.players)-1 {
				g.round++
				if g.round > 3 {
					goto done
				}
				fmt.Printf("------ Round %d -------\n", g.round)
			}

		}
	}

done:
	fmt.Printf("Ending game routine\n")
}

func (g *Game) NewRound(players []Player) ([]Player, error) {
	var playersOut []Player
	var card Card
	g.Action = make(chan Action, len(players))
	// first card
	for _, p := range players {
		p.Cards = make([]Card, 0)
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.Cards = append(p.Cards, card)
	}

	// second card
	for _, p := range players {
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.Cards = append(p.Cards, card)
		playersOut = append(playersOut, p)
	}
	g.players = playersOut
	g.round++
	go g.gameRoutine()
	return playersOut, nil
}

func (g *Game) NewPlayer(name string) Player {
	p := Player{Name: name}
	p.Plays = make(chan Play)
	p.game = g
	p.id = uuid.NewRandom()
	return p
}
