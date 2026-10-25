package dto

// CountryCode is an ISO 3166 alpha-2 country code accepted by ASL (doc 13.27).
type CountryCode string

// Country codes. UZ is Uzbekistan; use it with ReleaseProduction.
const (
	CountryAD CountryCode = "AD"
	CountryAE CountryCode = "AE"
	CountryAF CountryCode = "AF"
	CountryAG CountryCode = "AG"
	CountryAI CountryCode = "AI"
	CountryAL CountryCode = "AL"
	CountryAM CountryCode = "AM"
	CountryAO CountryCode = "AO"
	CountryAQ CountryCode = "AQ"
	CountryAR CountryCode = "AR"
	CountryAS CountryCode = "AS"
	CountryAT CountryCode = "AT"
	CountryAU CountryCode = "AU"
	CountryAW CountryCode = "AW"
	CountryAX CountryCode = "AX"
	CountryAZ CountryCode = "AZ"
	CountryBA CountryCode = "BA"
	CountryBB CountryCode = "BB"
	CountryBD CountryCode = "BD"
	CountryBE CountryCode = "BE"
	CountryBF CountryCode = "BF"
	CountryBG CountryCode = "BG"
	CountryBH CountryCode = "BH"
	CountryBI CountryCode = "BI"
	CountryBJ CountryCode = "BJ"
	CountryBL CountryCode = "BL"
	CountryBM CountryCode = "BM"
	CountryBN CountryCode = "BN"
	CountryBO CountryCode = "BO"
	CountryBQ CountryCode = "BQ"
	CountryBR CountryCode = "BR"
	CountryBS CountryCode = "BS"
	CountryBT CountryCode = "BT"
	CountryBV CountryCode = "BV"
	CountryBW CountryCode = "BW"
	CountryBY CountryCode = "BY"
	CountryBZ CountryCode = "BZ"
	CountryCA CountryCode = "CA"
	CountryCC CountryCode = "CC"
	CountryCD CountryCode = "CD"
	CountryCF CountryCode = "CF"
	CountryCG CountryCode = "CG"
	CountryCH CountryCode = "CH"
	CountryCI CountryCode = "CI"
	CountryCK CountryCode = "CK"
	CountryCL CountryCode = "CL"
	CountryCM CountryCode = "CM"
	CountryCN CountryCode = "CN"
	CountryCO CountryCode = "CO"
	CountryCR CountryCode = "CR"
	CountryCU CountryCode = "CU"
	CountryCV CountryCode = "CV"
	CountryCW CountryCode = "CW"
	CountryCX CountryCode = "CX"
	CountryCY CountryCode = "CY"
	CountryCZ CountryCode = "CZ"
	CountryDE CountryCode = "DE"
	CountryDJ CountryCode = "DJ"
	CountryDK CountryCode = "DK"
	CountryDM CountryCode = "DM"
	CountryDO CountryCode = "DO"
	CountryDZ CountryCode = "DZ"
	CountryEC CountryCode = "EC"
	CountryEE CountryCode = "EE"
	CountryEG CountryCode = "EG"
	CountryEH CountryCode = "EH"
	CountryER CountryCode = "ER"
	CountryES CountryCode = "ES"
	CountryET CountryCode = "ET"
	CountryEU CountryCode = "EU"
	CountryFI CountryCode = "FI"
	CountryFJ CountryCode = "FJ"
	CountryFK CountryCode = "FK"
	CountryFM CountryCode = "FM"
	CountryFO CountryCode = "FO"
	CountryFR CountryCode = "FR"
	CountryGA CountryCode = "GA"
	CountryGB CountryCode = "GB"
	CountryGD CountryCode = "GD"
	CountryGE CountryCode = "GE"
	CountryGF CountryCode = "GF"
	CountryGG CountryCode = "GG"
	CountryGH CountryCode = "GH"
	CountryGI CountryCode = "GI"
	CountryGL CountryCode = "GL"
	CountryGM CountryCode = "GM"
	CountryGN CountryCode = "GN"
	CountryGP CountryCode = "GP"
	CountryGQ CountryCode = "GQ"
	CountryGR CountryCode = "GR"
	CountryGS CountryCode = "GS"
	CountryGT CountryCode = "GT"
	CountryGU CountryCode = "GU"
	CountryGW CountryCode = "GW"
	CountryGY CountryCode = "GY"
	CountryHK CountryCode = "HK"
	CountryHM CountryCode = "HM"
	CountryHN CountryCode = "HN"
	CountryHR CountryCode = "HR"
	CountryHT CountryCode = "HT"
	CountryHU CountryCode = "HU"
	CountryID CountryCode = "ID"
	CountryIE CountryCode = "IE"
	CountryIL CountryCode = "IL"
	CountryIM CountryCode = "IM"
	CountryIN CountryCode = "IN"
	CountryIO CountryCode = "IO"
	CountryIQ CountryCode = "IQ"
	CountryIR CountryCode = "IR"
	CountryIS CountryCode = "IS"
	CountryIT CountryCode = "IT"
	CountryJE CountryCode = "JE"
	CountryJM CountryCode = "JM"
	CountryJO CountryCode = "JO"
	CountryJP CountryCode = "JP"
	CountryKE CountryCode = "KE"
	CountryKG CountryCode = "KG"
	CountryKH CountryCode = "KH"
	CountryKI CountryCode = "KI"
	CountryKM CountryCode = "KM"
	CountryKN CountryCode = "KN"
	CountryKP CountryCode = "KP"
	CountryKR CountryCode = "KR"
	CountryKW CountryCode = "KW"
	CountryKY CountryCode = "KY"
	CountryKZ CountryCode = "KZ"
	CountryLA CountryCode = "LA"
	CountryLB CountryCode = "LB"
	CountryLC CountryCode = "LC"
	CountryLI CountryCode = "LI"
	CountryLK CountryCode = "LK"
	CountryLR CountryCode = "LR"
	CountryLS CountryCode = "LS"
	CountryLT CountryCode = "LT"
	CountryLU CountryCode = "LU"
	CountryLV CountryCode = "LV"
	CountryLY CountryCode = "LY"
	CountryMA CountryCode = "MA"
	CountryMC CountryCode = "MC"
	CountryMD CountryCode = "MD"
	CountryME CountryCode = "ME"
	CountryMF CountryCode = "MF"
	CountryMG CountryCode = "MG"
	CountryMH CountryCode = "MH"
	CountryMK CountryCode = "MK"
	CountryML CountryCode = "ML"
	CountryMM CountryCode = "MM"
	CountryMN CountryCode = "MN"
	CountryMO CountryCode = "MO"
	CountryMP CountryCode = "MP"
	CountryMQ CountryCode = "MQ"
	CountryMR CountryCode = "MR"
	CountryMS CountryCode = "MS"
	CountryMT CountryCode = "MT"
	CountryMU CountryCode = "MU"
	CountryMV CountryCode = "MV"
	CountryMW CountryCode = "MW"
	CountryMX CountryCode = "MX"
	CountryMY CountryCode = "MY"
	CountryMZ CountryCode = "MZ"
	CountryNA CountryCode = "NA"
	CountryNC CountryCode = "NC"
	CountryNE CountryCode = "NE"
	CountryNF CountryCode = "NF"
	CountryNG CountryCode = "NG"
	CountryNI CountryCode = "NI"
	CountryNL CountryCode = "NL"
	CountryNO CountryCode = "NO"
	CountryNP CountryCode = "NP"
	CountryNR CountryCode = "NR"
	CountryNU CountryCode = "NU"
	CountryNZ CountryCode = "NZ"
	CountryOM CountryCode = "OM"
	CountryPA CountryCode = "PA"
	CountryPE CountryCode = "PE"
	CountryPF CountryCode = "PF"
	CountryPG CountryCode = "PG"
	CountryPH CountryCode = "PH"
	CountryPK CountryCode = "PK"
	CountryPL CountryCode = "PL"
	CountryPM CountryCode = "PM"
	CountryPN CountryCode = "PN"
	CountryPR CountryCode = "PR"
	CountryPS CountryCode = "PS"
	CountryPT CountryCode = "PT"
	CountryPW CountryCode = "PW"
	CountryPY CountryCode = "PY"
	CountryQA CountryCode = "QA"
	CountryRE CountryCode = "RE"
	CountryRO CountryCode = "RO"
	CountryRS CountryCode = "RS"
	CountryRU CountryCode = "RU"
	CountryRW CountryCode = "RW"
	CountrySA CountryCode = "SA"
	CountrySB CountryCode = "SB"
	CountrySC CountryCode = "SC"
	CountrySD CountryCode = "SD"
	CountrySE CountryCode = "SE"
	CountrySG CountryCode = "SG"
	CountrySH CountryCode = "SH"
	CountrySI CountryCode = "SI"
	CountrySJ CountryCode = "SJ"
	CountrySK CountryCode = "SK"
	CountrySL CountryCode = "SL"
	CountrySM CountryCode = "SM"
	CountrySN CountryCode = "SN"
	CountrySO CountryCode = "SO"
	CountrySR CountryCode = "SR"
	CountrySS CountryCode = "SS"
	CountryST CountryCode = "ST"
	CountrySV CountryCode = "SV"
	CountrySX CountryCode = "SX"
	CountrySY CountryCode = "SY"
	CountrySZ CountryCode = "SZ"
	CountryTC CountryCode = "TC"
	CountryTD CountryCode = "TD"
	CountryTF CountryCode = "TF"
	CountryTG CountryCode = "TG"
	CountryTH CountryCode = "TH"
	CountryTJ CountryCode = "TJ"
	CountryTK CountryCode = "TK"
	CountryTL CountryCode = "TL"
	CountryTM CountryCode = "TM"
	CountryTN CountryCode = "TN"
	CountryTO CountryCode = "TO"
	CountryTR CountryCode = "TR"
	CountryTT CountryCode = "TT"
	CountryTV CountryCode = "TV"
	CountryTW CountryCode = "TW"
	CountryTZ CountryCode = "TZ"
	CountryUA CountryCode = "UA"
	CountryUG CountryCode = "UG"
	CountryUM CountryCode = "UM"
	CountryUS CountryCode = "US"
	CountryUY CountryCode = "UY"
	CountryUZ CountryCode = "UZ"
	CountryVA CountryCode = "VA"
	CountryVC CountryCode = "VC"
	CountryVE CountryCode = "VE"
	CountryVG CountryCode = "VG"
	CountryVI CountryCode = "VI"
	CountryVN CountryCode = "VN"
	CountryVU CountryCode = "VU"
	CountryWF CountryCode = "WF"
	CountryWS CountryCode = "WS"
	CountryYE CountryCode = "YE"
	CountryYT CountryCode = "YT"
	CountryZA CountryCode = "ZA"
	CountryZM CountryCode = "ZM"
	CountryZW CountryCode = "ZW"
)

// countryCodes is the set of all known country codes.
var countryCodes = map[CountryCode]struct{}{
	CountryAD: {},
	CountryAE: {},
	CountryAF: {},
	CountryAG: {},
	CountryAI: {},
	CountryAL: {},
	CountryAM: {},
	CountryAO: {},
	CountryAQ: {},
	CountryAR: {},
	CountryAS: {},
	CountryAT: {},
	CountryAU: {},
	CountryAW: {},
	CountryAX: {},
	CountryAZ: {},
	CountryBA: {},
	CountryBB: {},
	CountryBD: {},
	CountryBE: {},
	CountryBF: {},
	CountryBG: {},
	CountryBH: {},
	CountryBI: {},
	CountryBJ: {},
	CountryBL: {},
	CountryBM: {},
	CountryBN: {},
	CountryBO: {},
	CountryBQ: {},
	CountryBR: {},
	CountryBS: {},
	CountryBT: {},
	CountryBV: {},
	CountryBW: {},
	CountryBY: {},
	CountryBZ: {},
	CountryCA: {},
	CountryCC: {},
	CountryCD: {},
	CountryCF: {},
	CountryCG: {},
	CountryCH: {},
	CountryCI: {},
	CountryCK: {},
	CountryCL: {},
	CountryCM: {},
	CountryCN: {},
	CountryCO: {},
	CountryCR: {},
	CountryCU: {},
	CountryCV: {},
	CountryCW: {},
	CountryCX: {},
	CountryCY: {},
	CountryCZ: {},
	CountryDE: {},
	CountryDJ: {},
	CountryDK: {},
	CountryDM: {},
	CountryDO: {},
	CountryDZ: {},
	CountryEC: {},
	CountryEE: {},
	CountryEG: {},
	CountryEH: {},
	CountryER: {},
	CountryES: {},
	CountryET: {},
	CountryEU: {},
	CountryFI: {},
	CountryFJ: {},
	CountryFK: {},
	CountryFM: {},
	CountryFO: {},
	CountryFR: {},
	CountryGA: {},
	CountryGB: {},
	CountryGD: {},
	CountryGE: {},
	CountryGF: {},
	CountryGG: {},
	CountryGH: {},
	CountryGI: {},
	CountryGL: {},
	CountryGM: {},
	CountryGN: {},
	CountryGP: {},
	CountryGQ: {},
	CountryGR: {},
	CountryGS: {},
	CountryGT: {},
	CountryGU: {},
	CountryGW: {},
	CountryGY: {},
	CountryHK: {},
	CountryHM: {},
	CountryHN: {},
	CountryHR: {},
	CountryHT: {},
	CountryHU: {},
	CountryID: {},
	CountryIE: {},
	CountryIL: {},
	CountryIM: {},
	CountryIN: {},
	CountryIO: {},
	CountryIQ: {},
	CountryIR: {},
	CountryIS: {},
	CountryIT: {},
	CountryJE: {},
	CountryJM: {},
	CountryJO: {},
	CountryJP: {},
	CountryKE: {},
	CountryKG: {},
	CountryKH: {},
	CountryKI: {},
	CountryKM: {},
	CountryKN: {},
	CountryKP: {},
	CountryKR: {},
	CountryKW: {},
	CountryKY: {},
	CountryKZ: {},
	CountryLA: {},
	CountryLB: {},
	CountryLC: {},
	CountryLI: {},
	CountryLK: {},
	CountryLR: {},
	CountryLS: {},
	CountryLT: {},
	CountryLU: {},
	CountryLV: {},
	CountryLY: {},
	CountryMA: {},
	CountryMC: {},
	CountryMD: {},
	CountryME: {},
	CountryMF: {},
	CountryMG: {},
	CountryMH: {},
	CountryMK: {},
	CountryML: {},
	CountryMM: {},
	CountryMN: {},
	CountryMO: {},
	CountryMP: {},
	CountryMQ: {},
	CountryMR: {},
	CountryMS: {},
	CountryMT: {},
	CountryMU: {},
	CountryMV: {},
	CountryMW: {},
	CountryMX: {},
	CountryMY: {},
	CountryMZ: {},
	CountryNA: {},
	CountryNC: {},
	CountryNE: {},
	CountryNF: {},
	CountryNG: {},
	CountryNI: {},
	CountryNL: {},
	CountryNO: {},
	CountryNP: {},
	CountryNR: {},
	CountryNU: {},
	CountryNZ: {},
	CountryOM: {},
	CountryPA: {},
	CountryPE: {},
	CountryPF: {},
	CountryPG: {},
	CountryPH: {},
	CountryPK: {},
	CountryPL: {},
	CountryPM: {},
	CountryPN: {},
	CountryPR: {},
	CountryPS: {},
	CountryPT: {},
	CountryPW: {},
	CountryPY: {},
	CountryQA: {},
	CountryRE: {},
	CountryRO: {},
	CountryRS: {},
	CountryRU: {},
	CountryRW: {},
	CountrySA: {},
	CountrySB: {},
	CountrySC: {},
	CountrySD: {},
	CountrySE: {},
	CountrySG: {},
	CountrySH: {},
	CountrySI: {},
	CountrySJ: {},
	CountrySK: {},
	CountrySL: {},
	CountrySM: {},
	CountrySN: {},
	CountrySO: {},
	CountrySR: {},
	CountrySS: {},
	CountryST: {},
	CountrySV: {},
	CountrySX: {},
	CountrySY: {},
	CountrySZ: {},
	CountryTC: {},
	CountryTD: {},
	CountryTF: {},
	CountryTG: {},
	CountryTH: {},
	CountryTJ: {},
	CountryTK: {},
	CountryTL: {},
	CountryTM: {},
	CountryTN: {},
	CountryTO: {},
	CountryTR: {},
	CountryTT: {},
	CountryTV: {},
	CountryTW: {},
	CountryTZ: {},
	CountryUA: {},
	CountryUG: {},
	CountryUM: {},
	CountryUS: {},
	CountryUY: {},
	CountryUZ: {},
	CountryVA: {},
	CountryVC: {},
	CountryVE: {},
	CountryVG: {},
	CountryVI: {},
	CountryVN: {},
	CountryVU: {},
	CountryWF: {},
	CountryWS: {},
	CountryYE: {},
	CountryYT: {},
	CountryZA: {},
	CountryZM: {},
	CountryZW: {},
}

// IsValid reports whether the code is in the ASL country list.
func (c CountryCode) IsValid() bool {
	_, ok := countryCodes[c]
	return ok
}
