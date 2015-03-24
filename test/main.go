package main

import (
	"fmt"

	"github.com/porjo/gopoke"
)

func main() {

	var err error
	var players []gopoke.Player

	game := gopoke.NewGame()

	p1 := game.NewPlayer("bob")
	p2 := game.NewPlayer("jane")

	players = append(players, p1, p2)
	players, err = game.NewRound(players)
	if err != nil {
		fmt.Printf("New round failed, %s\n", err)
		return
	}

	fmt.Printf("players  %+v\n", players)
}
