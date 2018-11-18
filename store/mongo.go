package store

import (
	"fmt"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
)

type State string

const (
	Created     State = "created"
	Active      State = "active"
	Interrupted State = "interrupted"
	Finished    State = "finished"
)

type Stream struct {
	State     State
	CreatedAt time.Time `bson:"createdAt"`
}

func (stream *Stream) SetState(state State) error {
	curState := stream.State
	isAllowed := false

	switch curState {
	case Created:
		switch state {
		case Active:
			isAllowed = true
		}
	case Active:
		switch state {
		case Interrupted, Finished:
			isAllowed = true
		}
	case Interrupted:
		switch state {
		case Active, Finished:
			isAllowed = true
		}
	case Finished:
	}

	if !isAllowed {
		return fmt.Errorf("could not change stream state: transition is not allowed (current state: %s)", curState)
	}

	stream.State = state
	return nil
}

type Mongo struct {
	conn *mongo.Connection
}

var ConcurrentError = fmt.Errorf("could not change stream state: race detected: state changed middle place")

func NewMongo(conn *mongo.Connection) *Mongo {
	return &Mongo{conn}
}

func (m *Mongo) GetStream(id string) (stream Stream, err error) {
	err = m.conn.WithCollection(func(coll *mgo.Collection) error {
		query := bson.M{"_id": bson.ObjectIdHex(id)}
		return coll.Find(query).One(&stream)
	})
	return stream, err
}

func (m *Mongo) SetState(id string, state State) (stream Stream, err error) {
	stream, err = m.GetStream(id)
	if err != nil {
		// stream is not found
		return stream, err
	}

	prevState := stream.State
	err = stream.SetState(state)
	if err != nil {
		// transition is not allowed
		return stream, err
	}

	// save changes in store
	err = m.conn.WithCollection(func(coll *mgo.Collection) error {
		change := mgo.Change{
			Update:    bson.M{"state": state},
			ReturnNew: true,
		}
		_, err := coll.Find(bson.M{"_id": bson.ObjectIdHex(id), "state": prevState}).Apply(change, &stream)
		if err != nil && err == mgo.ErrNotFound {
			return ConcurrentError
		}
		return err
	})
	return stream, err
}

func (m *Mongo) SetStateAdmin(id string, state State) error {
	return m.conn.WithCollection(func(coll *mgo.Collection) error {
		change := mgo.Change{
			Update: bson.M{"state": state},
		}
		_, err := coll.Find(bson.M{"_id": bson.ObjectIdHex(id)}).Apply(change, nil)
		return err
	})
}
