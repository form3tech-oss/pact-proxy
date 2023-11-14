package pactproxy

import (
	"sync"
)

type Interactions struct {
	interactions sync.Map
}

func (i *Interactions) Store(interaction *Interaction) {
	i.interactions.Store(interaction.Description, interaction)
	if interaction.Alias != "" {
		i.interactions.Store(interaction.Alias, interaction)
	}
}

func (i *Interactions) Clear() {
	i.interactions.Range(func(k, _ interface{}) bool {
		v, _ := i.interactions.Load(k)
		v.(*Interaction).doneChannel <- struct{}{}
		i.interactions.Delete(k)
		return true
	})
}

func (i *Interactions) Load(key string) (*Interaction, bool) {
	result, ok := i.interactions.Load(key)
	if !ok {
		return nil, false
	}
	return result.(*Interaction), true
}

func (i *Interactions) FindAll(path, method string) ([]*Interaction, bool) {
	interactions := make(map[string]*Interaction)
	var result []*Interaction
	i.interactions.Range(func(_, v interface{}) bool {
		if v.(*Interaction).Match(path, method) {
			i := v.(*Interaction)
			interactions[i.Description] = i
		}
		return true
	})

	for _, i := range interactions {
		result = append(result, i)
	}

	return result, len(result) > 0
}

func (i *Interactions) All() []*Interaction {
	var interactions []*Interaction
	i.interactions.Range(func(_, v interface{}) bool {
		interactions = append(interactions, v.(*Interaction))
		return true
	})
	return interactions
}

func (i *Interactions) AllHaveRequests() bool {
	result := true
	i.interactions.Range(func(_, v interface{}) bool {
		if !v.(*Interaction).HasRequests(1) {
			result = false
			return false
		}
		return true
	})
	return result
}
