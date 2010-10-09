package main

type EntityID int32

type Entity struct {
	EntityID EntityID
}

type EntityManager struct {
	nextEntityID EntityID
	entities     map[EntityID]*Entity
}

// Allocate and assign a new entity ID
func (mgr *EntityManager) AddEntity(entity *Entity) {
	// EntityManager starts initialized to zero
	if mgr.entities == nil {
		mgr.entities = make(map[EntityID]*Entity)
	}

	// Search for next free ID
	entityID := mgr.nextEntityID
	_, exists := mgr.entities[entityID]
	for exists {
		entityID++
		if entityID == mgr.nextEntityID {
			panic("EntityID space exhausted")
		}
		_, exists = mgr.entities[entityID]
	}

	entity.EntityID = entityID
	mgr.entities[entityID] = entity
	mgr.nextEntityID = entityID + 1
}

func (mgr *EntityManager) RemoveEntity(entity *Entity) {
	mgr.entities[entity.EntityID] = nil, false
}
