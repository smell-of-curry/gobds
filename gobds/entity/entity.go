package entity

// Entity ...
type Entity struct {
	runtimeID uint64
	actorType string
}

// NewEntity ...
func NewEntity(runtimeID uint64, actorType string) Entity {
	return Entity{
		runtimeID: runtimeID,
		actorType: actorType,
	}
}

// RuntimeID ...
func (e Entity) RuntimeID() uint64 {
	return e.runtimeID
}

// ActorType ...
func (e Entity) ActorType() string {
	return e.actorType
}
