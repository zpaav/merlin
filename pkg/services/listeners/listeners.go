// Merlin is a post-exploitation command and control framework.
// This file is part of Merlin.
// Copyright (C) 2023  Russel Van Tuyl

// Merlin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.

// Merlin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Merlin.  If not, see <http://www.gnu.org/licenses/>.

// Package listeners is a service for creating and managing Listener objects
package listeners

import (
	// Standard
	"fmt"
	"sort"
	"strings"

	// 3rd Party
	uuid "github.com/satori/go.uuid"

	// Merlin
	"github.com/Ne0nd0g/merlin/pkg/listeners"
	"github.com/Ne0nd0g/merlin/pkg/listeners/http"
	httpMemory "github.com/Ne0nd0g/merlin/pkg/listeners/http/memory"
	"github.com/Ne0nd0g/merlin/pkg/listeners/smb"
	smbMemory "github.com/Ne0nd0g/merlin/pkg/listeners/smb/memory"
	"github.com/Ne0nd0g/merlin/pkg/listeners/tcp"
	tcpMemory "github.com/Ne0nd0g/merlin/pkg/listeners/tcp/memory"
	"github.com/Ne0nd0g/merlin/pkg/listeners/udp"
	udpMemory "github.com/Ne0nd0g/merlin/pkg/listeners/udp/memory"
	"github.com/Ne0nd0g/merlin/pkg/logging"
	"github.com/Ne0nd0g/merlin/pkg/servers"
	httpServer "github.com/Ne0nd0g/merlin/pkg/servers/http"
	httpServerRepo "github.com/Ne0nd0g/merlin/pkg/servers/http/memory"
)

// ListenerService is a structure that implements the service methods holding references to Listener & Server repositories
type ListenerService struct {
	httpRepo       http.Repository
	httpServerRepo httpServer.Repository
	smbRepo        smb.Repository
	tcpRepo        tcp.Repository
	udpRepo        udp.Repository
}

// NewListenerService is a factory to create and return a ListenerService
func NewListenerService() (ls ListenerService) {
	ls.httpRepo = WithHTTPMemoryListenerRepository()
	ls.httpServerRepo = WithHTTPMemoryServerRepository()
	ls.smbRepo = WithSMBMemoryListenerRepository()
	ls.tcpRepo = WithTCPMemoryListenerRepository()
	ls.udpRepo = WithUDPMemoryListenerRepository()
	return
}

// WithHTTPMemoryListenerRepository retrieves an in-memory HTTP Listener repository interface used to manage Listener objects
func WithHTTPMemoryListenerRepository() http.Repository {
	return httpMemory.NewRepository()
}

// WithHTTPMemoryServerRepository retrieves an in-memory HTTP Server repository interface used to manage Server objects
func WithHTTPMemoryServerRepository() httpServer.Repository {
	return httpServerRepo.NewRepository()
}

// WithSMBMemoryListenerRepository retrieves an in-memory TCP Listener repository interface used to manage Listener objects
func WithSMBMemoryListenerRepository() smb.Repository {
	return smbMemory.NewRepository()
}

// WithTCPMemoryListenerRepository retrieves an in-memory TCP Listener repository interface used to manage Listener objects
func WithTCPMemoryListenerRepository() tcp.Repository {
	return tcpMemory.NewRepository()
}

func WithUDPMemoryListenerRepository() udp.Repository {
	return udpMemory.NewRepository()
}

// NewListener is a factory that takes in a map of options used to configure a Listener, adds the Listener to its
// respective repository, and returns a copy created Listener object
func (ls *ListenerService) NewListener(options map[string]string) (listener listeners.Listener, er error) {
	// Determine the infrastructure layer server
	if _, ok := options["Protocol"]; !ok {
		return nil, fmt.Errorf("pkg/services/listeners.NewListener(): the options map did not contain the \"Protocol\" key")
	}

	switch strings.ToLower(options["Protocol"]) {
	//case servers.HTTP, servers.HTTPS, servers.H2C, servers.HTTP2, servers.HTTP3:
	case "http", "https", "h2c", "http2", "http3":
		hServer, err := httpServer.New(options)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		err = ls.httpServerRepo.Add(hServer)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		// Create a new HTTP Listener
		hListener, err := http.NewHTTPListener(&hServer, options)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		// Store the HTTP Listener
		err = ls.httpRepo.Add(hListener)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		logging.Server(fmt.Sprintf("Created %s listener on %s with name: %s, ID %s, Authenticator: %s, Transforms: %s", hServer.ProtocolString(), hServer.Addr(), hListener.Name(), hListener.ID(), hListener.Authenticator(), hListener.Transformers()))
		listener = &hListener
		return
	case "smb":
		// Create a new SMB Listener
		sListener, err := smb.NewSMBListener(options)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		// Store the SMB Listener
		err = ls.smbRepo.Add(sListener)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		logging.Server(fmt.Sprintf("Created %s listener on %s with name: %s, ID: %s, Authenticator: %s, Transforms: %s", options["Protocol"], sListener.Addr(), sListener.Name(), sListener.ID(), sListener.Authenticator(), sListener.Transformers()))
		listener = &sListener
		return
	case "tcp":
		// Create a new TCP Listener
		tListener, err := tcp.NewTCPListener(options)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		// Store the TCP Listener
		err = ls.tcpRepo.Add(tListener)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		logging.Server(fmt.Sprintf("Created %s listener on %s with name: %s, ID: %s, Authenticator: %s, Transforms: %s", options["Protocol"], tListener.Addr(), tListener.Name(), tListener.ID(), tListener.Authenticator(), tListener.Transformers()))
		listener = &tListener
		return
	case "udp":
		uListener, err := udp.NewUDPListener(options)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		// Store the TCP Listener
		err = ls.udpRepo.Add(uListener)
		if err != nil {
			return nil, fmt.Errorf("pkg/services/listeners.NewListener(): %s", err)
		}
		logging.Server(fmt.Sprintf("Created %s listener on %s with name: %s, ID: %s, Authenticator: %s, Transforms: %s", options["Protocol"], uListener.Addr(), uListener.Name(), uListener.ID(), uListener.Authenticator(), uListener.Transformers()))
		listener = &uListener
		return
	default:
		return nil, fmt.Errorf("pkg/services/listeners.NewListener(): unhandled server type %d", servers.FromString(options["Protocol"]))
	}
}

// CLICompleter returns a list of Listener & Server types that Merlin supports for CLI tab completion
func (ls *ListenerService) CLICompleter() func(string) []string {
	return func(line string) []string {
		var s []string
		l := listeners.Listeners()
		for _, listener := range l {
			switch listener {
			case listeners.HTTP:
				srvs := servers.RegisteredServers
				for k := range srvs {
					s = append(s, servers.Protocol(k))
				}
			default:
				s = append(s, listeners.String(listener))
			}
		}
		return s
	}
}

// DefaultOptions gets the default configurable options for both the listener and the infrastructure layer server (if applicable)
func (ls *ListenerService) DefaultOptions(protocol string) (options map[string]string, err error) {
	var listenerOptions map[string]string
	var serverOptions map[string]string
	switch listeners.FromString(protocol) {
	case listeners.HTTP:
		// Listener options
		listenerOptions = http.DefaultOptions()
		// Server, infrastructure layer, options
		serverOptions = httpServer.GetDefaultOptions(servers.FromString(protocol))
	case listeners.SMB:
		listenerOptions = smb.DefaultOptions()
	case listeners.TCP:
		listenerOptions = tcp.DefaultOptions()
	case listeners.UDP:
		listenerOptions = udp.DefaultOptions()
	default:
		err = fmt.Errorf("pkg/services/listeners.DefaultOptions(): unhandled server type: %s", protocol)
		return
	}

	// Add Server options (if any) to Listener options
	for k, v := range serverOptions {
		listenerOptions[k] = v
	}

	// Sort the keys
	var keys []string
	for key, _ := range listenerOptions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	options = make(map[string]string, len(listenerOptions))
	for _, key := range keys {
		options[key] = listenerOptions[key]
	}
	return
}

// List returns a list of Listener names that exist and is used for command line tab completion
func (ls *ListenerService) List() func(string) []string {
	return func(line string) []string {
		return ls.ListenerNames()
	}
}

// Listener returns a Listener object for the input ID
func (ls *ListenerService) Listener(id uuid.UUID) (listeners.Listener, error) {
	httpListener, err := ls.httpRepo.ListenerByID(id)
	if err == nil {
		return &httpListener, nil
	}
	smbListener, err := ls.smbRepo.ListenerByID(id)
	if err == nil {
		return &smbListener, nil
	}
	tcpListener, err := ls.tcpRepo.ListenerByID(id)
	if err == nil {
		return &tcpListener, nil
	}
	udpListener, err := ls.udpRepo.ListenerByID(id)
	if err == nil {
		return &udpListener, nil
	}
	return nil, fmt.Errorf("pkg/services/listeners.GetListenerByID(): could not find listener %s", id)
}

// Listeners returns a list of stored Listener objects
func (ls *ListenerService) Listeners() (listenerList []listeners.Listener) {
	// HTTP Listeners
	httpListeners := ls.httpRepo.Listeners()
	for i, _ := range httpListeners {
		listenerList = append(listenerList, &httpListeners[i])
	}
	// SMB Listeners
	smbListeners := ls.smbRepo.Listeners()
	for i, _ := range smbListeners {
		listenerList = append(listenerList, &smbListeners[i])
	}
	// TCP Listeners
	tcpListeners := ls.tcpRepo.Listeners()
	for i, _ := range tcpListeners {
		listenerList = append(listenerList, &tcpListeners[i])
	}
	// UDP Listeners
	udpListeners := ls.udpRepo.Listeners()
	for i, _ := range udpListeners {
		listenerList = append(listenerList, &udpListeners[i])
	}
	return
}

// ListenerNames returns a list of Listener names as a string
func (ls *ListenerService) ListenerNames() (names []string) {
	// HTTP Listeners
	httpListeners := ls.httpRepo.Listeners()
	for _, listener := range httpListeners {
		names = append(names, listener.Name())
	}
	// SMB Listeners
	smbListeners := ls.smbRepo.Listeners()
	for _, listener := range smbListeners {
		names = append(names, listener.Name())
	}
	// TCP Listeners
	tcpListeners := ls.tcpRepo.Listeners()
	for _, listener := range tcpListeners {
		names = append(names, listener.Name())
	}
	// UDP Listeners
	udpListeners := ls.udpRepo.Listeners()
	for _, listener := range udpListeners {
		names = append(names, listener.Name())
	}
	return
}

// ListenerByName returns the first Listener object that matches the input name
func (ls *ListenerService) ListenerByName(name string) (listeners.Listener, error) {
	listener, err := ls.httpRepo.ListenerByName(name)
	if err == nil {
		return &listener, err
	}
	smbListener, err := ls.smbRepo.ListenerByName(name)
	if err == nil {
		return &smbListener, err
	}
	tcpListener, err := ls.tcpRepo.ListenerByName(name)
	if err == nil {
		return &tcpListener, err
	}
	udpListener, err := ls.udpRepo.ListenerByName(name)
	if err == nil {
		return &udpListener, err
	}
	return nil, fmt.Errorf("pkg/services/listeners.GetListenerByName(): %s", err)
}

// ListenersByType returns a list of all stored listeners for the provided listener
func (ls *ListenerService) ListenersByType(protocol int) (listenerList []listeners.Listener) {
	switch protocol {
	case listeners.HTTP:
		httpListeners := ls.httpRepo.Listeners()
		for _, listener := range httpListeners {
			listenerList = append(listenerList, &listener)
		}
	case listeners.SMB:
		smbListeners := ls.smbRepo.Listeners()
		for _, listener := range smbListeners {
			listenerList = append(listenerList, &listener)
		}
	case listeners.TCP:
		tcpListeners := ls.tcpRepo.Listeners()
		for _, listener := range tcpListeners {
			listenerList = append(listenerList, &listener)
		}
	case listeners.UDP:
		udpListeners := ls.udpRepo.Listeners()
		for _, listener := range udpListeners {
			listenerList = append(listenerList, &listener)
		}
	}
	return
}

// Remove deletes the Listener from its repository
func (ls *ListenerService) Remove(id uuid.UUID) error {
	listener, err := ls.Listener(id)
	if err != nil {
		return err
	}

	switch listener.Protocol() {
	case listeners.HTTP:
		return ls.httpRepo.RemoveByID(id)
	case listeners.SMB:
		return ls.smbRepo.RemoveByID(id)
	case listeners.TCP:
		return ls.tcpRepo.RemoveByID(id)
	case listeners.UDP:
		return ls.udpRepo.RemoveByID(id)
	default:
		return fmt.Errorf("pkg/services/listeners.Remove(): unhandled listener protocol type %d for listener %s", listener.Protocol(), id)
	}
}

// Restart terminates a Listener's embedded Server object (if applicable) and then starts it again
func (ls *ListenerService) Restart(id uuid.UUID) error {
	// Get the listener
	listener, err := ls.Listener(id)
	if err != nil {
		return fmt.Errorf("pkg/services/listeners.Restart(): %s", err)
	}
	server := *listener.Server()
	err = server.Stop()
	if err != nil {
		return fmt.Errorf("pkg/services/listeners.Restart(): %s", err)
	}
	go server.Start()
	return nil
}

// SetOption updates an existing Listener's configurable option with the value provided
func (ls *ListenerService) SetOption(id uuid.UUID, option, value string) error {
	listener, err := ls.Listener(id)
	if err != nil {
		return err
	}
	switch listener.Protocol() {
	case listeners.HTTP:
		return ls.httpRepo.SetOption(id, option, value)
	case listeners.SMB:
		return ls.smbRepo.SetOption(id, option, value)
	case listeners.TCP:
		return ls.tcpRepo.SetOption(id, option, value)
	case listeners.UDP:
		return ls.udpRepo.SetOption(id, option, value)
	default:
		return fmt.Errorf("pkg/services/listeners.SetOptions(): unhandled protocol %d for listener %s", listener.Protocol(), id)
	}
}

// Start initiates the Listener's embedded Server object (if applicable) to start listening and responding to Agent communications
func (ls *ListenerService) Start(id uuid.UUID) error {
	// Get the listener
	listener, err := ls.Listener(id)
	if err != nil {
		return fmt.Errorf("pkg/services/listeners.Start(): %s", err)
	}
	switch listener.Protocol() {
	case listeners.HTTP:
		server := *listener.Server()
		// Start() does not return until the transport server is killed and therefore must be run in a go routine
		go server.Start()
		return nil
	case listeners.SMB:
		return nil
	case listeners.TCP:
		// Nothing to do, there is not an infrastructure layer server to start for the TCP listener
		return nil
	case listeners.UDP:
		return nil
	default:
		return fmt.Errorf("pkg/services/listeners.Start(): unhandled listener protocol: %d", listener.Protocol())
	}
}

// Stop terminates the Listener's embedded Server object (if applicable) to stop it listening for incoming Agent messages
func (ls *ListenerService) Stop(id uuid.UUID) error {
	// Get the listener
	listener, err := ls.Listener(id)
	if err != nil {
		return fmt.Errorf("pkg/services/listeners.Restart(): %s", err)
	}
	if listener.Protocol() == listeners.HTTP {
		server := *listener.Server()
		return server.Stop()
	}
	return nil
}
