package render

import (
	"math"
	"sort"
)

const (
	campMaxFigures        = 20
	campForegroundFigures = 12
)

type campPerson struct {
	position [4]float64 // x, base y, perspective scale, facing direction
	pose     int
	standing bool
}

func campFigureCount(population uint32) int {
	return int(min(uint32(campMaxFigures), population))
}

func campCrowd(population uint32) []campPerson {
	count := campFigureCount(population)
	people := make([]campPerson, 0, count)
	// Fill opposite sides in pairs, then extend the seated ring to twelve.
	seats := [campForegroundFigures][4]float64{
		{446, 423, .86, 1}, {545, 394, .76, 1}, {737, 397, .79, -1}, {843, 430, .90, -1},
		{343, 494, 1.15, 1}, {944, 491, 1.12, -1}, {486, 551, 1.27, 1}, {795, 551, 1.30, -1},
		{402, 460, .99, 1}, {884, 461, .99, -1}, {575, 575, .98, 1}, {710, 575, .98, -1},
	}
	for _, index := range [campForegroundFigures]int{0, 3, 1, 2, 4, 5, 6, 7, 8, 9, 10, 11} {
		if len(people) == count {
			break
		}
		people = append(people, campPerson{position: seats[index], pose: index})
	}
	// Up to eight standing, front-facing figures fill the background.
	// Every pose remains inside the gathering's existing animation crop.
	columns := count - len(people)
	for column := range columns {
		fraction := float64(column+1) / float64(columns+1)
		x := 310 + 660*fraction
		y := 390 - 10*math.Sin(fraction*math.Pi)
		people = append(people, campPerson{position: [4]float64{x, y, .62, 0}, pose: campForegroundFigures + column, standing: true})
	}
	sort.Slice(people, func(i, j int) bool { return people[i].position[1] < people[j].position[1] })
	return people
}
