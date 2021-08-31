package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Color int

const (
	Empty Color = iota
	Blue
	Red
	Black
	White
	Green
	Yellow
)

func elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", what, time.Since(start))
	}
}

const Npegs = 4
const Ncolors = int(Yellow + 1)

type Guess [Npegs]Color

var ColorNames = []string{"empty", "blue", "red", "black", "white", "green", "yellow"}

var ColorLetters = map[rune]Color{'e': Empty, 'b': Blue, 'r': Red, 'k': Black,
	'w': White, 'g': Green, 'y': Yellow}

type Status int

const (
	Notfound Status = iota
	Found
	Impossible
)

func (c Color) String() string {
	return ColorNames[c]
}

func RandomGuess() (guess Guess) {
	for i := range guess {
		guess[i] = Color(rand.Intn(Ncolors))
	}
	return guess
}

func GuessFromString(s string) (guess Guess) {
	i := 0 // separate counter in the case there might be non-ASCII chars in s
	for _, c := range s {
		guess[i] = ColorLetters[c]
		i++
	}
	return guess
}

type Answer struct {
	blacks int
	whites int
}

type Fact struct {
	guess  Guess
	answer Answer
}

const frequencyListSize = (Npegs + 1) * (Npegs + 1)

type FrequencyList [frequencyListSize]int

func Index(answer Answer) int {
	return (Npegs+1)*answer.blacks + answer.whites
}

func (fl FrequencyList) Incr(answer Answer) {
	fl[Index(answer)] += 1
}

func (f Fact) String() string {
	return fmt.Sprintf("[%v %v]", f.guess, f.answer)
}

func Allows(fact Fact, guess Guess) bool {
	return fact.answer == Compare(fact.guess, guess)
}

func AllowsAll(facts []Fact, guess Guess) bool {
	for _, fact := range facts {
		if !Allows(fact, guess) {
			return false
		}
	}
	return true
}

func (a Answer) String() string {
	return fmt.Sprintf("[blacks=%d whites=%d]", a.blacks, a.whites)
}

func Compare(g1 Guess, g2 Guess) Answer {
	blacks, whites := 0, 0
	var used1, used2 [Npegs]bool
	for i := 0; i < Npegs; i++ {
		if g1[i] == g2[i] {
			blacks += 1
			used1[i], used2[i] = true, true
		}
	}
	for i := 0; i < Npegs; i++ {
		for j := 0; j < Npegs; j++ {
			if i != j && !used1[i] && !used2[j] && g1[i] == g2[j] {
				whites += 1
				used1[i], used2[j] = true, true
			}
		}
	}
	return Answer{blacks, whites}
}

func CountAllBlacks(fl FrequencyList) int {
	return fl[Index(Answer{Npegs, 0})]
}

// computes the informational value (aka Shannon entropy)
// corresponding to the frequency list
// see https://en.wikipedia.org/wiki/Entropy_(information_theory)
func InfoValue(fl FrequencyList) float64 {
	r := 0.0
	ntot := 0.0
	for _, v := range fl {
		ntot += 1
		if v != 0 && v != 1 {
			r -= float64(v) * math.Log(float64(v))
		}
	}
	if ntot > 0 {
		return (r/ntot + math.Log(ntot)) / math.Log(2.0)
	} else {
		return 0.0
	}
}

func calculateAllGuessesInner(result *[]Guess, prevColors []Color) {
	if len(prevColors) == Npegs {
		var prevColorsArray [Npegs]Color
		copy(prevColorsArray[:], prevColors)
		*result = append(*result, Guess(prevColorsArray))
	} else {
		for i := 0; i < Ncolors; i++ {
			calculateAllGuessesInner(result, append(prevColors, Color(i)))
		}
	}
}

func calculateAllGuesses() []Guess {
	var result []Guess
	defer elapsed("calculate all guesses")()
	calculateAllGuessesInner(&result, []Color{})
	return result
}

var allGuesses = calculateAllGuesses()

var fixedFirstGuess = Guess{Yellow, Blue, Red, Black}

const randomFirstGuess = false

func firstGuess() Guess {
	if !randomFirstGuess {
		return fixedFirstGuess
	}
	var guess Guess
	if Ncolors >= Npegs {
		guessColors := rand.Perm(Ncolors)[0:Npegs]
		for i, c := range guessColors {
			guess[i] = Color(c)
		}
	} else {
		guess = RandomGuess()
	}
	return guess
}

func MakeGuess(facts []Fact) (Guess, Status, int) {
	defer elapsed("make guess")()
	var guess Guess
	if len(facts) == 0 {
		guess = firstGuess()
		return guess, Notfound, int(math.Pow(float64(Ncolors), float64(Npegs)))
	}
	var possibleSolutions []Guess
	for _, guess := range allGuesses {
		if AllowsAll(facts, guess) {
			possibleSolutions = append(possibleSolutions, guess)
		}
	}
	if len(possibleSolutions) == 1 {
		return possibleSolutions[0], Found, 1
	} else if len(possibleSolutions) == 0 {
		return guess, Impossible, 0
	}
	infoValues := make([]float64, len(allGuesses))
	var wg sync.WaitGroup
	wg.Add(len(allGuesses))
	for i := range allGuesses {
		go func(i int) {
			defer wg.Done()
			fl := FrequencyList{}
			for _, solution := range possibleSolutions {
				answer := Compare(allGuesses[i], solution)
				fl.Incr(answer)
			}
			infoValues[i] = InfoValue(fl)
		}(i)
	}
	wg.Wait()
	maxInfoValue := infoValues[0]
	for _, v := range infoValues {
		if v > maxInfoValue {
			maxInfoValue = v
		}
	}
	maxIdx := 0
	for idx, guess := range allGuesses {
		if infoValues[idx] == maxInfoValue {
			if AllowsAll(facts, guess) {
				return guess, Notfound, len(possibleSolutions)
			} else {
				maxIdx = idx
			}
		}
	}
	return allGuesses[maxIdx], Notfound, len(possibleSolutions)
}

func AllBlacks(a Answer) bool {
	return a.blacks == Npegs
}

func Play(ask func(Guess) (Answer, bool)) {
	var facts []Fact
	var guess Guess
	state := Notfound
	counter := 0
	for state == Notfound {
		var possibilities int
		guess, state, possibilities = MakeGuess(facts)
		fmt.Printf("%d possible solutions left.\n", possibilities)
		if state == Impossible {
			break
		}
		answer, quit := ask(guess)
		if quit {
			fmt.Println("quitting")
			return
		} else if AllBlacks(answer) {
			state = Found
		} else {
			facts = append(facts, Fact{guess, answer})
		}
		counter += 1
	}
	if state == Found {
		fmt.Printf("Found solution: %v in %d steps.\n", guess, counter)
	} else {
		fmt.Println("No solution possible.")
	}
}

func PlayAutomatically(solution Guess) {
	defer elapsed("play automatically")()
	ask := func(guess Guess) (Answer, bool) {
		answer := Compare(guess, solution)
		fmt.Printf("%v %v\n", guess, answer)
		return answer, false
	}
	Play(ask)
}

func PlayManually() {
	fmt.Printf("Please think of a color for %d pegs.\n", Npegs)
	fmt.Printf("Possible colors are %v.\n", ColorNames)
	fmt.Printf("For each question, respond with the number of black pegs and white pegs, ")
	fmt.Printf("separated by a space.\n")
	fmt.Printf("Enter 'q' to quit.\n")
	ask := func(guess Guess) (Answer, bool) {
		fmt.Println()
		fmt.Printf("My guess is %v\n", guess)
		fmt.Println("Black pegs, white pegs?")
		var blacks, whites int
		_, err := fmt.Scanf("%d %d\n", &blacks, &whites)
		if err != nil {
			return Answer{}, true
		}
		return Answer{blacks, whites}, false
	}
	Play(ask)
}

func benchmarkCompare(g1, g2 Guess) {
	defer elapsed("benchmark compare")()
	for i := 0; i < 1000000; i++ {
		_ = Compare(g1, g2)
	}
}

func benchmarkFL(g1, g2 Guess) {
	defer elapsed("benchmark getting element from frequency list")()
	answer := Compare(g1, g2)
	fl := FrequencyList{}
	for i := 0; i < 1000000; i++ {
		fl.Incr(answer)
	}
}

func main() {
	fmt.Println("*** Play Automatically ***")
	PlayAutomatically(Guess{Red, White, Green, Empty})

	fmt.Println("\n*** Play Manually ***")
	PlayManually()

	fmt.Println("\n*** Benchmarks ***")
	benchmarkCompare(RandomGuess(), RandomGuess())
	benchmarkFL(RandomGuess(), RandomGuess())
}
