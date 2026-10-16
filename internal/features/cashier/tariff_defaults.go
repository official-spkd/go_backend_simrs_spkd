package cashier

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

// Tariff codes the bill derives its lines from.
const (
	TariffRegistration        = "ADM01"
	TariffGeneralConsultation = "KON01"
	TariffSpecialConsultation = "KON02"
)

// defaultTariffs are seeded once; prices edited later in the database are kept. Names are
// printed on the patient's invoice, so they are in Indonesian.
func defaultTariffs() []models.Tariff {
	effective := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	mk := func(code, name string, general, bpjs, insurance int64) models.Tariff {
		return models.Tariff{Code: code, Name: name, PriceGeneral: general, PriceBPJS: bpjs, PriceInsurance: insurance, EffectiveDate: effective, Version: 1}
	}
	return []models.Tariff{
		mk(TariffRegistration, "Karcis rawat jalan", 30000, 0, 35000),
		mk(TariffGeneralConsultation, "Konsultasi dokter umum", 75000, 0, 90000),
		mk(TariffSpecialConsultation, "Konsultasi dokter spesialis", 150000, 0, 175000),
	}
}
