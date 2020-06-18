package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func toEvent(input interface{}) Event {
	// TODO: why does the line not work?
	// return input.(Event)

	jsonString, _ := json.Marshal(input)
	// convert json to struct
	cast1 := Event{}
	json.Unmarshal(jsonString, &cast1)

	return cast1
}

func TestSequentialPlayback(t *testing.T) {
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	// --- Set up recording
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	// --- Load cassette and replay: matching on sequence
	replayer, err := NewReplayer(filepath, sequentialComparator)

	fmt.Println("Replaying...")
	// feed the requests in *backwards* order, but responses come back in original order
	replayedResp1, err1 := replayer.Write(s2)
	replayedResp2, err2 := replayer.Write(s1)

	castResp1 := toEvent(replayedResp1)
	castResp2 := toEvent(replayedResp2)

	if err1 != nil {
		fmt.Println(err1)
	} else {
		fmt.Println(replayedResp1)
	}
	assert.Equal(t, castResp1.Name, r1.Name)
	assert.Equal(t, castResp1.Id, r1.Id)

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Println(castResp2)
	}

	assert.Equal(t, castResp2.Name, r2.Name)
	assert.Equal(t, castResp2.Id, r2.Id)
}

func TestFirstMatchingEvent(t *testing.T) {
	firstMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		// TODO: do we really have to Marshal then Unmarshal? This just feels wrong :/
		// convert map to json
		cast1 := toEvent(req1)
		cast2 := req2.(Event)
		return (cast1.Name == cast2.Name), true
	}

	// -- Set up recorder
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	// --- Replay - returning the first match
	replayer, err := NewReplayer(filepath, firstMatchingComparator)

	fmt.Println("Replaying...")

	// Send an unrecognized request
	_, err2 := replayer.Write(Event{})
	assert.EqualError(t, err2, "No matching events.")

	// Send the requests in opposite order, responses should come in opposite order
	replayedResp2, err2 := replayer.Write(s2)
	replayedResp1, err1 := replayer.Write(s1)

	castResp1 := toEvent(replayedResp1)
	castResp2 := toEvent(replayedResp2)

	assert.NoError(t, err2)
	assert.NoError(t, err1)
	assert.Equal(t, castResp1, r1)
	assert.Equal(t, castResp2, r2)
}

func TestLastMatchingEvent(t *testing.T) {
	lastMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		jsonString, _ := json.Marshal(req1)
		// convert json to struct
		cast1 := Event{}
		json.Unmarshal(jsonString, &cast1)

		cast2 := req2.(Event)
		return (cast1.Name == cast2.Name), false // false to return last match
	}

	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 1", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	s3 := Event{Name: "Event 3", Id: 52}
	r3 := Event{Name: "Response 3", Id: 52}
	fmt.Printf("%+v | %+v\n", s3, r3)
	recorder.Write(s3, r3)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(filepath, lastMatchingComparator)

	fmt.Println("Replaying...")
	fmt.Println("Should return last matching event to \"Event 1\"")
	respA, errA := replayer.Write(s1)
	castA := toEvent(respA)

	check(errA)
	assert.Equal(t, castA.Name, r2.Name)
	assert.Equal(t, castA.Id, r2.Id)

	fmt.Println("Should match the single \"Event 3\"")
	respB, errB := replayer.Write(s3)
	castB := toEvent(respB)
	check(errB)
	assert.Equal(t, castB.Name, r3.Name)
	assert.Equal(t, castB.Id, r3.Id)

	fmt.Println("Should return first matching event to \"Event 1\" - since the last one was removed")
	respC, errC := replayer.Write(s1)
	castC := toEvent(respC)
	check(errC)
	assert.Equal(t, castC.Name, r1.Name)
	assert.Equal(t, castC.Id, r1.Id)
	// TODO: fix this
	// replayedResp1 = replayedResp1.(Event)
	// replayedResp2 = replayedResp2.(Event)

	// assert.Equal(t, replayedResp1, r1)
	// assert.Equal(t, replayedResp2, r2)
}
