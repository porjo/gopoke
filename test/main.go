package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/porjo/gopoke"
)

var game *gopoke.Game
var wg sync.WaitGroup

func main() {
	var err error
	var players []gopoke.Player

	rand.Seed(time.Now().UTC().UnixNano())

	game = gopoke.NewGame()

	game.NewPlayer("bob")
	game.NewPlayer("jane")
	game.NewPlayer("max")

	players, err = game.Start()
	if err != nil {
		fmt.Printf("Start game failed, %s\n", err)
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

	fmt.Printf("%s: entering loop\n", p.Name())

	tickChan := time.NewTicker(time.Millisecond * 1000).C

	for {
		select {
		case play := <-p.GamePlay:

			if play.Player != nil {
				fmt.Printf("%s: receive game play %s (%s)\n", p.Name(), play.Action, play.Player.Name())
			} else {
				fmt.Printf("%s: receive notify my turn\n", p.Name())
			}
			if len(play.ValidActions) > 0 {
				idxs := rand.Perm(len(play.ValidActions))

				newplay := gopoke.Play{}
				newplay.Player = p
				newplay.Action = play.ValidActions[idxs[0]]
				newplay.Amount = 15
				fmt.Printf("%s: sending play %s\n", p.Name(), newplay.Action)
				game.PlayerPlay <- newplay
			}
		case <-tickChan:
			fmt.Printf("%s: tick\n", p.Name())

		}
	}
}
