package account

import (
	"fmt"
	"github.com/go-foreman/examples/pkg/sagas/usecase"
	"github.com/go-foreman/examples/pkg/sagas/usecase/account/contracts"
	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	systemContacts "github.com/go-foreman/foreman/saga/contracts"
)

func init() {
	scheme.KnownTypesRegistryInstance.AddKnownTypes(contracts.AccountGroup, &RegisterAccountSaga{})
	usecase.DefaultSagasCollection.AddSaga(&RegisterAccountSaga{})
}

type RegisterAccountSaga struct {
	saga.BaseSaga
	UID                 string `json:"uid"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	RetriesLimit        int    `json:"retries_limit"`
	CurrentStage        string `json:"current_stage"`
}

func (r *RegisterAccountSaga) Init() {
	r.
		AddEventHandler(&contracts.AccountRegistered{}, r.AccountRegistered).
		AddEventHandler(&contracts.RegistrationFailed{}, r.RegistrationFailed).
		AddEventHandler(&contracts.ConfirmationSendingFailed{}, r.ConfirmationSendingFailed).
		AddEventHandler(&contracts.AccountConfirmed{}, r.AccountConfirmed)
}

func (r *RegisterAccountSaga) Start(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Starting saga %s", execCtx.SagaInstance().UID()))
	execCtx.Dispatch(&contracts.RegisterAccountCmd{
		UID:   r.UID,
		Email: r.Email,
	})
	return nil
}

func (r *RegisterAccountSaga) Compensate(execCtx saga.SagaContext) error {
	return nil
}

func (r *RegisterAccountSaga) Recover(execCtx saga.SagaContext) error {
	r.RetriesLimit = 1
	if ev := execCtx.SagaInstance().Status().FailedOnEvent(); ev != nil {
		execCtx.Dispatch(ev)
	}
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Recovering saga, Retries limit was set to 1"))

	return nil
}

func (r *RegisterAccountSaga) AccountRegistered(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account registration successful. Sending confirmation to %s", r.Email))
	execCtx.Dispatch(&contracts.SendConfirmationCmd{
		UID:   r.UID,
		Email: r.Email,
	})
	return nil
}

func (r *RegisterAccountSaga) RegistrationFailed(execCtx saga.SagaContext) error {
	msg, _ := execCtx.Message().Payload().(*contracts.RegistrationFailed)

	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Account registration failed. Reason %s", msg.Reason))

	if r.RetriesLimit > 0 {
		r.RetriesLimit--
		execCtx.Dispatch(&contracts.RegisterAccountCmd{
			UID:   r.UID,
			Email: r.Email,
		})
		return nil
	}

	execCtx.SagaInstance().Fail(execCtx.Message().Payload())
	execCtx.Dispatch(&systemContacts.RecoverSagaCommand{SagaUID: execCtx.SagaInstance().UID()})
	return nil
}

func (r *RegisterAccountSaga) ConfirmationSendingFailed(execCtx saga.SagaContext) error {
	msg, _ := execCtx.Message().Payload().(*contracts.ConfirmationSendingFailed)

	//some retry logic if you want
	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Failed sending account confirmation for %s. Reason %s", r.Email, msg.Reason))
	execCtx.SagaInstance().Fail(execCtx.Message().Payload())

	return nil
}

func (r *RegisterAccountSaga) AccountConfirmed(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account %s confirmed by %s", r.UID, r.Email))

	execCtx.SagaInstance().Complete()
	return nil
}
