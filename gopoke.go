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

	go g.gameRoutine()

	return g
}

func (g *Game) gameRoutine() error {

	// start game
	select {
	case a := <-g.Action:
		fmt.Printf("game action %v\n", a)
	}
	return nil
}

func (g *Game) NewRound(players []Player) ([]Player, error) {
	var playersOut []Player
	var card Card
	if g.round > 3 {
		return nil, fmt.Errorf("last round has been played, game over")
	}
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
	return playersOut, nil
}

func (g *Game) NewPlayer(name string) Player {
	p := Player{Name: name}
	p.game = g
	p.id = uuid.NewRandom()
	return p
}
