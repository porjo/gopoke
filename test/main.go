package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/porjo/gopoke"
)

var game *gopoke.Game
var wg sync.WaitGroup

func main() {

	var err error
	var players []gopoke.Player

	game = gopoke.NewGame()

	p1 := game.NewPlayer("bob")
	p2 := game.NewPlayer("jane")
	p3 := game.NewPlayer("max")

	players = append(players, p1, p2, p3)
	players, err = game.NewRound(players)
	if err != nil {
		fmt.Printf("New round failed, %s\n", err)
		return
	}

	for i, _ := range players {
		wg.Add(1)
		go playerRoutine(&players[i])
	}

	fmt.Printf("waiting for players...\n")
	wg.Wait()
}

func playerRoutine(p *gopoke.Player) {

	defer wg.Done()

	fmt.Printf("%s: entering loop\n", p.Name)

	tickChan := time.NewTicker(time.Millisecond * 1000).C

	for {
		select {
		case play := <-p.Plays:
			fmt.Printf("%s: receive game play %v\n", p.Name, play)
			if len(play.ValidActions) > 0 {
				c := gopoke.Check{Player: p}
				fmt.Printf("%s: sending action %v\n", p.Name, c)
				game.Action <- c
			}
		case <-tickChan:
			fmt.Printf("%s: tick\n", p.Name)

		}
	}
}
