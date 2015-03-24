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
	Action  chan Action
	Pot     int
	players []Player
	cards   []Card
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
	game := &Game{}

	for _, suit := range suits {
		for _, val := range suitVals {
			card := Card{Suit: suit, Value: val}
			game.cards = append(game.cards, card)
		}
	}

	var shuffled []Card
	for _, idx := range rand.Perm(len(game.cards)) {
		shuffled = append(shuffled, game.cards[idx])
	}

	game.cards = shuffled

	go game.gameRoutine()

	return game
}

func (g *Game) gameRoutine() error {

	return nil
}

func (g *Game) NewRound(players []Player) ([]Player, error) {
	var playersOut []Player
	var card Card
	for _, p := range players {
		if p.Chips == 0 {
			return nil, fmt.Errorf("player %s chips cannot be $0", p.Name)
		}
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.Cards = append(p.Cards, card)
		card, g.cards = g.cards[len(g.cards)-1], g.cards[:len(g.cards)-1]
		p.Cards = append(p.Cards, card)
		playersOut = append(playersOut, p)
	}
	g.players = playersOut
	return playersOut, nil
}

func (g *Game) NewPlayer(name string) Player {
	p := Player{Name: name}
	p.game = g
	p.id = uuid.NewRandom()
	return p
}
