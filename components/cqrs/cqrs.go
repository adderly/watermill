package cqrs

import (
	stdErrors "errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// Deprecated: use CommandProcessor and EventProcessor instead.
type FacadeConfig struct {
	// GenerateCommandsTopic generates topic name based on the command name.
	// Command name is generated by CommandEventMarshaler's Name method.
	//
	// It allows you to use topic per command or one topic for every command.
	GenerateCommandsTopic func(commandName string) string

	// CommandHandlers return command handlers which should be executed.
	CommandHandlers func(commandBus *CommandBus, eventBus *EventBus) []CommandHandler

	// CommandsPublisher is Publisher used to publish commands.
	CommandsPublisher message.Publisher

	// CommandsSubscriberConstructor is constructor for subscribers which will subscribe for messages.
	// It will be called for every command handler.
	// It allows you to create separated customized Subscriber for every command handler.
	CommandsSubscriberConstructor CommandsSubscriberConstructor

	// GenerateEventsTopic generates topic name based on the event name.
	// Event name is generated by CommandEventMarshaler's Name method.
	//
	// It allows you to use topic per command or one topic for every command.
	GenerateEventsTopic func(eventName string) string

	// EventHandlers return event handlers which should be executed.
	EventHandlers func(commandBus *CommandBus, eventBus *EventBus) []EventHandler

	// EventsPublisher is Publisher used to publish commands.
	EventsPublisher message.Publisher

	// EventsSubscriberConstructor is constructor for subscribers which will subscribe for messages.
	// It will be called for every event handler.
	// It allows you to create separated customized Subscriber for every event handler.
	EventsSubscriberConstructor EventsSubscriberConstructor

	// Router is a Watermill router, which will be used to handle events and commands.
	// Router handlers will be automatically generated by AddHandlersToRouter of Command and Event handlers.
	Router *message.Router

	CommandEventMarshaler CommandEventMarshaler

	Logger watermill.LoggerAdapter
}

func (c FacadeConfig) Validate() error {
	var err error

	if c.CommandsEnabled() {
		if c.GenerateCommandsTopic == nil {
			err = stdErrors.Join(err, errors.New("GenerateCommandsTopic is nil"))
		}
		if c.CommandsSubscriberConstructor == nil {
			err = stdErrors.Join(err, errors.New("CommandsSubscriberConstructor is nil"))
		}
		if c.CommandsPublisher == nil {
			err = stdErrors.Join(err, errors.New("CommandsPublisher is nil"))
		}
	}
	if c.EventsEnabled() {
		if c.GenerateEventsTopic == nil {
			err = stdErrors.Join(err, errors.New("GenerateEventsTopic is nil"))
		}
		if c.EventsSubscriberConstructor == nil {
			err = stdErrors.Join(err, errors.New("EventsSubscriberConstructor is nil"))
		}
		if c.EventsPublisher == nil {
			err = stdErrors.Join(err, errors.New("EventsPublisher is nil"))
		}
	}

	if c.Router == nil {
		err = stdErrors.Join(err, errors.New("Router is nil"))
	}
	if c.Logger == nil {
		err = stdErrors.Join(err, errors.New("Logger is nil"))
	}
	if c.CommandEventMarshaler == nil {
		err = stdErrors.Join(err, errors.New("CommandEventMarshaler is nil"))
	}

	return err
}

func (c FacadeConfig) EventsEnabled() bool {
	return c.GenerateEventsTopic != nil || c.EventsPublisher != nil || c.EventsSubscriberConstructor != nil
}

func (c FacadeConfig) CommandsEnabled() bool {
	return c.GenerateCommandsTopic != nil || c.CommandsPublisher != nil || c.CommandsSubscriberConstructor != nil
}

// Deprecated: use CommandHandler and EventHandler instead.
//
// Facade is a facade for creating the Command and Event buses and processors.
// It was created to avoid boilerplate, when using CQRS in the standard way.
// You can also create buses and processors manually, drawing inspiration from how it's done in NewFacade.
type Facade struct {
	commandsTopic func(commandName string) string
	commandBus    *CommandBus

	eventsTopic func(eventName string) string
	eventBus    *EventBus

	commandEventMarshaler CommandEventMarshaler
}

func (f Facade) CommandBus() *CommandBus {
	return f.commandBus
}

func (f Facade) EventBus() *EventBus {
	return f.eventBus
}

func (f Facade) CommandEventMarshaler() CommandEventMarshaler {
	return f.commandEventMarshaler
}

// Deprecated: use CommandHandler and EventHandler instead.
func NewFacade(config FacadeConfig) (*Facade, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	c := &Facade{
		commandsTopic:         config.GenerateCommandsTopic,
		eventsTopic:           config.GenerateEventsTopic,
		commandEventMarshaler: config.CommandEventMarshaler,
	}

	if config.CommandsEnabled() {
		var err error
		c.commandBus, err = NewCommandBus(
			config.CommandsPublisher,
			config.GenerateCommandsTopic,
			config.CommandEventMarshaler,
		)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create command bus")
		}
	} else {
		config.Logger.Info("Empty GenerateCommandsTopic, command bus will be not created", nil)
	}
	if config.EventsEnabled() {
		var err error
		c.eventBus, err = NewEventBus(config.EventsPublisher, config.GenerateEventsTopic, config.CommandEventMarshaler)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create event bus")
		}
	} else {
		config.Logger.Info("Empty GenerateEventsTopic, event bus will be not created", nil)
	}

	if config.CommandHandlers != nil {
		commandProcessor, err := NewCommandProcessor(
			config.CommandHandlers(c.commandBus, c.eventBus),
			config.GenerateCommandsTopic,
			config.CommandsSubscriberConstructor,
			config.CommandEventMarshaler,
			config.Logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create command processor")
		}

		if err := commandProcessor.AddHandlersToRouter(config.Router); err != nil {
			return nil, err
		}
	}
	if config.EventHandlers != nil {
		eventProcessor, err := NewEventProcessor(
			config.EventHandlers(c.commandBus, c.eventBus),
			config.GenerateEventsTopic,
			config.EventsSubscriberConstructor,
			config.CommandEventMarshaler,
			config.Logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create event processor")
		}

		if err := eventProcessor.AddHandlersToRouter(config.Router); err != nil {
			return nil, err
		}
	}

	return c, nil
}
