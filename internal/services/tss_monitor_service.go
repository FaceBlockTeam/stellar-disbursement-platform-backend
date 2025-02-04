package services

import (
	"context"
	"fmt"

	"github.com/stellar/go/support/log"
	"github.com/stellar/stellar-disbursement-platform-backend/internal/data"
	"github.com/stellar/stellar-disbursement-platform-backend/internal/db"
	txSubStore "github.com/stellar/stellar-disbursement-platform-backend/internal/transactionsubmission/store"
)

type TSSMonitorService struct {
	sdpModels *data.Models
	tssModel  *txSubStore.TransactionModel
}

// MonitorTransactions monitors TSS transactions and updates payments accordingly.
func (s TSSMonitorService) MonitorTransactions(ctx context.Context, batchSize int) error {
	err := db.RunInTransaction(ctx, s.sdpModels.DBConnectionPool, nil, func(dbTx db.DBTransaction) error {
		return s.monitorTransactions(ctx, dbTx, batchSize)
	})
	if err != nil {
		return fmt.Errorf("error sending payments: %w", err)
	}

	return nil
}

// monitorTransactions monitors TSS transactions and updates payments accordingly.
func (s TSSMonitorService) monitorTransactions(ctx context.Context, dbTx db.DBTransaction, batchSize int) error {
	// 1. Get transactions that are in a final state (status=SUCCESS or status=ERROR)
	//     this operation will lock the rows.
	transactions, err := s.tssModel.GetTransactionBatchForUpdate(ctx, dbTx, batchSize)
	if err != nil {
		return fmt.Errorf("error getting transactions for update: %w", err)
	}
	if len(transactions) == 0 {
		log.Ctx(ctx).Infof("No transactions to sync")
		return nil
	}

	// 2. Split transactions into successful and failed
	failedTransactions := []*txSubStore.Transaction{}
	successfulTransactions := []*txSubStore.Transaction{}
	for _, transaction := range transactions {
		if !transaction.StellarTransactionHash.Valid {
			return fmt.Errorf("expected transaction %s to have a stellar transaction hash", transaction.ID)
		}
		if transaction.Status == txSubStore.TransactionStatusSuccess {
			successfulTransactions = append(successfulTransactions, transaction)
		} else if transaction.Status == txSubStore.TransactionStatusError {
			failedTransactions = append(failedTransactions, transaction)
		} else {
			return fmt.Errorf("transaction id %s is in an unexpected status: %s", transaction.ID, transaction.Status)
		}
	}

	// 3. Update payments based on the status of the transactions
	if len(successfulTransactions) > 0 {
		log.Ctx(ctx).Infof("Syncing payments for %d successful transactions", len(successfulTransactions))
		errPayments := s.syncPaymentsWithTransactions(ctx, dbTx, successfulTransactions, data.SuccessPaymentStatus)
		if errPayments != nil {
			return fmt.Errorf("error syncing payments for successful transactions: %w", errPayments)
		}
	}
	if len(failedTransactions) > 0 {
		log.Ctx(ctx).Infof("Syncing payments for %d failed transactions", len(failedTransactions))
		errPayments := s.syncPaymentsWithTransactions(ctx, dbTx, failedTransactions, data.FailedPaymentStatus)
		if errPayments != nil {
			return fmt.Errorf("error syncing payments for failed transactions: %w", errPayments)
		}
	}

	// 4. Set synced_at for all synced transactions
	transactionIDs := make([]string, len(transactions))
	for i, transaction := range transactions {
		transactionIDs[i] = transaction.ID
	}
	err = s.tssModel.UpdateSyncedTransactions(ctx, dbTx, transactionIDs)
	if err != nil {
		return fmt.Errorf("error updating transactions as synced: %w", err)
	}

	return nil
}

// syncPaymentsWithTransactions updates the status of the payments based on the status of the transactions.
func (s TSSMonitorService) syncPaymentsWithTransactions(ctx context.Context, dbTx db.DBTransaction, transactions []*txSubStore.Transaction, toStatus data.PaymentStatus) error {
	paymentIDs := make([]string, len(transactions))
	for i, transaction := range transactions {
		paymentIDs[i] = transaction.ExternalID
	}
	payments, errPayments := s.sdpModels.Payment.GetByIDs(ctx, dbTx, paymentIDs)
	if errPayments != nil {
		return fmt.Errorf("error getting payments by ids: %w", errPayments)
	}

	// Create a map of disbursement id from payment
	disbursementMap := make(map[string]struct{}, len(payments))
	paymentMap := make(map[string]*data.Payment, len(payments))

	for _, payment := range payments {
		if payment.Status != data.PendingPaymentStatus {
			return fmt.Errorf("error getting payments by ids: expected payment %s to be in pending status but got %s", payment.ID, payment.Status)
		}
		paymentMap[payment.ID] = payment
		disbursementMap[payment.Disbursement.ID] = struct{}{}
	}

	// Update payment status for each transaction to SUCCESS or FAILURE
	for _, transaction := range transactions {
		payment := paymentMap[transaction.ExternalID]
		if payment == nil {
			// The payment associated with this transaction was deleted.
			log.Ctx(ctx).Errorf("orphaned transaction - Unable to sync transaction %s because the associated payment %s was deleted",
				transaction.ID,
				transaction.ExternalID)
			continue
		}
		paymentUpdate := &data.PaymentUpdate{
			Status:               toStatus,
			StatusMessage:        transaction.StatusMessage.String,
			StellarTransactionID: transaction.StellarTransactionHash.String,
		}
		errUpdate := s.sdpModels.Payment.Update(ctx, dbTx, payment, paymentUpdate)
		if errUpdate != nil {
			return fmt.Errorf("error updating payment id %s for transaction id %s: %w", payment.ID, transaction.ID, errUpdate)
		}
	}

	disbursementIDs := make([]string, 0, len(disbursementMap))
	for disbursement := range disbursementMap {
		disbursementIDs = append(disbursementIDs, disbursement)
	}
	err := s.sdpModels.Disbursements.CompleteDisbursements(ctx, dbTx, disbursementIDs)
	if err != nil {
		return fmt.Errorf("error completing disbursement: %w", err)
	}

	return nil
}

// NewTSSMonitorService creates a new TSSMonitorService instance.
func NewTSSMonitorService(models *data.Models) *TSSMonitorService {
	return &TSSMonitorService{
		sdpModels: models,
		tssModel:  txSubStore.NewTransactionModel(models.DBConnectionPool),
	}
}
