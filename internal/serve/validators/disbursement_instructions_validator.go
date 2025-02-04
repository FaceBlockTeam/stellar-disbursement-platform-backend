package validators

import (
	"fmt"
	"strings"
	"time"

	"github.com/stellar/stellar-disbursement-platform-backend/internal/utils"

	"github.com/stellar/stellar-disbursement-platform-backend/internal/data"
)

type DisbursementInstructionsValidator struct {
	verificationField data.VerificationField
	*Validator
}

func NewDisbursementInstructionsValidator(verificationField data.VerificationField) *DisbursementInstructionsValidator {
	return &DisbursementInstructionsValidator{
		verificationField: verificationField,
		Validator:         NewValidator(),
	}
}

func (iv *DisbursementInstructionsValidator) ValidateInstruction(instruction *data.DisbursementInstruction, lineNumber int) {
	phone := strings.TrimSpace(instruction.Phone)
	id := strings.TrimSpace(instruction.ID)
	amount := strings.TrimSpace(instruction.Amount)
	verification := strings.TrimSpace(instruction.VerificationValue)

	// validate phone field
	iv.CheckError(utils.ValidatePhoneNumber(phone), fmt.Sprintf("line %d - phone", lineNumber), "invalid phone format. Correct format: +380445555555")
	iv.Check(strings.TrimSpace(phone) != "", fmt.Sprintf("line %d - phone", lineNumber), "phone cannot be empty")

	// validate id field
	iv.Check(strings.TrimSpace(id) != "", fmt.Sprintf("line %d - id", lineNumber), "id cannot be empty")

	// validate amount field
	iv.CheckError(utils.ValidateAmount(amount), fmt.Sprintf("line %d - amount", lineNumber), "invalid amount. Amount must be a positive number")

	// validate verification field
	// date of birth with format 2006-01-02
	if iv.verificationField == data.VerificationFieldDateOfBirth {
		_, err := time.Parse("2006-01-02", verification)
		iv.CheckError(err, fmt.Sprintf("line %d - birthday", lineNumber), "invalid date of birth format. Correct format: 1990-01-01")
	}
}
