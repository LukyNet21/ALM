package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const dateFmt = "02.01.2006"
const fileName = "denik.json"

// ArrayList

const baseCapacity = 4

type ArrayList struct {
	data     []string
	size     int
	capacity int
}

func newArrayList() *ArrayList {
	return &ArrayList{
		data:     make([]string, baseCapacity),
		capacity: baseCapacity,
	}
}

func (l *ArrayList) Add(s string) {
	if l.size >= l.capacity {
		l.capacity *= 2
		n := make([]string, l.capacity)
		for i := 0; i < l.size; i++ {
			n[i] = l.data[i]
		}
		l.data = n
	}
	l.data[l.size] = s
	l.size++
}

func (l *ArrayList) Get(i int) string {
	return l.data[i]
}

func (l *ArrayList) Remove(i int) {
	for j := i; j < l.size-1; j++ {
		l.data[j] = l.data[j+1]
	}
	l.data[l.size-1] = ""
	l.size--
}

func (l *ArrayList) Size() int {
	return l.size
}

// Den + vlastní spojový seznam dnů

type Day struct {
	date       time.Time
	entries    *ArrayList
	prev, next *Day
}

type Diary struct {
	head, tail, current *Day
	count               int
}

func (d *Diary) insertAfter(after *Day, date time.Time) *Day {
	node := &Day{date: date, entries: newArrayList()}
	if d.head == nil {
		d.head, d.tail = node, node
	} else {
		node.prev, node.next = after, after.next
		if after.next != nil {
			after.next.prev = node
		} else {
			d.tail = node
		}
		after.next = node
	}
	d.count++
	return node
}

func (d *Diary) removeDay(day *Day) {
	if day.prev != nil {
		day.prev.next = day.next
	} else {
		d.head = day.next
	}
	if day.next != nil {
		day.next.prev = day.prev
	} else {
		d.tail = day.prev
	}
	d.count--
}

// Vstup / výstup

var sc = bufio.NewScanner(os.Stdin)

func readLine() string {
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}

func readMultiline() string {
	fmt.Println("Text (prázdný řádek = konec):")
	text := ""
	for sc.Scan() && sc.Text() != "" {
		if text != "" {
			text += "\n"
		}
		text += sc.Text()
	}
	return text
}

func render(d *Diary, msg string) {
	fmt.Print("\033[H\033[2J")
	if msg != "" {
		fmt.Println(">>", msg)
	}
	if d.current == nil {
		fmt.Println("Deník je prázdný.\nOperace: novy | nacti | zavri")
	} else {
		c := d.current
		fmt.Printf("Dnů: %d | Datum: %s | Záznamů: %d\n\n", d.count, c.date.Format(dateFmt), c.entries.Size())
		for i := 0; i < c.entries.Size(); i++ {
			fmt.Printf("[%d] %s\n\n", i+1, c.entries.Get(i))
		}
		fmt.Println("Operace: predchozi | dalsi | zacatek | konec | novy | smaz | uloz | nacti | zavri")
	}
	fmt.Print("> ")
}

// Operace

func cmdNovy(d *Diary) string {
	var t time.Time
	var err error
	for {
		fmt.Print("Datum (dd.MM.yyyy): ")
		if t, err = time.Parse(dateFmt, readLine()); err == nil {
			break
		}
		fmt.Println("Chybný formát.")
	}

	text := readMultiline()
	if text == "" {
		return "Prázdný záznam nebyl přidán."
	}

	c := d.current
	switch {
	case c != nil && t.Equal(c.date):
		c.entries.Add(text)
		return "Záznam přidán."
	case c == nil || t.After(c.date) && (c.next == nil || t.Before(c.next.date)):
		d.current = d.insertAfter(c, t)
		d.current.entries.Add(text)
		return "Nový den vytvořen."
	}
	return "Zápis odmítnut."
}

func cmdSmaz(d *Diary) string {
	c := d.current
	if c == nil {
		return "Není co mazat."
	}
	fmt.Printf("Číslo záznamu (1-%d): ", c.entries.Size())
	num, err := strconv.Atoi(readLine())
	if err != nil || num < 1 || num > c.entries.Size() {
		return "Neplatné číslo."
	}

	fmt.Printf("%s\nOpravdu smazat? (a/n): ", c.entries.Get(num-1))
	if readLine() != "a" {
		return "Mazání zrušeno."
	}

	c.entries.Remove(num - 1)
	if c.entries.Size() == 0 {
		d.current = c.next
		if d.current == nil {
			d.current = c.prev
		}
		d.removeDay(c)
	}
	return "Záznam smazán."
}

// JSON save / load

type jsonDay struct {
	Date    time.Time `json:"date"`
	Entries []string  `json:"entries"`
}

type jsonFile struct {
	Count int       `json:"count"`
	Days  []jsonDay `json:"days"`
}

func cmdUloz(d *Diary) string {
	f := jsonFile{Count: d.count, Days: make([]jsonDay, d.count)}
	i := 0
	for n := d.head; n != nil; n = n.next {
		f.Days[i] = jsonDay{n.date, n.entries.data[:n.entries.size]}
		i++
	}
	data, _ := json.Marshal(f)
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return "Chyba: " + err.Error()
	}
	return "Uloženo."
}

func cmdNacti(d *Diary) string {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return "Chyba: " + err.Error()
	}

	var f jsonFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "Chyba: " + err.Error()
	}

	*d = Diary{}
	for i := 0; i < f.Count; i++ {
		node := d.insertAfter(d.tail, f.Days[i].Date)
		for _, e := range f.Days[i].Entries {
			node.entries.Add(e)
		}
	}
	d.current = d.head
	return "Načteno."
}

// main

func main() {
	d := &Diary{}
	msg := ""
	for {
		render(d, msg)
		msg = ""
		switch readLine() {
		case "predchozi":
			if d.current != nil && d.current.prev != nil {
				d.current = d.current.prev
			}
		case "dalsi":
			if d.current != nil && d.current.next != nil {
				d.current = d.current.next
			}
		case "zacatek":
			d.current = d.head
		case "konec":
			d.current = d.tail
		case "novy":
			msg = cmdNovy(d)
		case "smaz":
			msg = cmdSmaz(d)
		case "uloz":
			msg = cmdUloz(d)
		case "nacti":
			msg = cmdNacti(d)
		case "zavri":
			return
		default:
			msg = "Neznámá operace."
		}
	}
}
