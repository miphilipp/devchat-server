package websocket

import (
	//"fmt"
	"encoding/json"
)

func (s *Server) processLivePatchMessage(wrapper messageFrame, clientID int, msg json.RawMessage) error {
	_, err := s.Messaging.EditMessage(clientID, wrapper.Source, msg)
	if err != nil {
		return err
	}
	s.BroadcastToRoom(wrapper.Source, wrapper.Command, msg, wrapper.ID)
	return nil
}

func (s *Server) processPatchMessage(wrapper messageFrame, clientID int, msg json.RawMessage) error {
	messageID, err := s.Messaging.EditMessage(clientID, wrapper.Source, msg)
	if err != nil {
		return err
	}

	payload := struct {
		MessageID int `json:"messageId"`
	}{messageID}
	s.BroadcastToRoom(wrapper.Source, wrapper.Command, payload, wrapper.ID)
	return nil
}

func (s *Server) processNewMessage(wrapper messageFrame, clientID int, msg json.RawMessage) error {
	answer, err := s.Messaging.SendMessage(wrapper.Source, clientID, msg)
	if err != nil {
		return err
	}
	s.BroadcastToRoom(wrapper.Source, wrapper.Command, answer, wrapper.ID)
	return nil
}

func (s *Server) processToggleLiveCodingSession(wrapper messageFrame, clientID int, msg json.RawMessage, state bool) error {
	messageID, err := s.Messaging.ToggleLiveSession(clientID, wrapper.Source, state, msg)
	if err != nil {
		return err
	}

	reply := struct {
		MessageID int `json:"messageId"`
		NewOwner  int `json:"newOwner"`
	}{messageID, clientID}

	s.BroadcastToRoom(wrapper.Source, wrapper.Command, reply, wrapper.ID)
	return nil
}
