/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package pcmiler

type ErrorCodes int

const (
	ErrorCodeOK                           = ErrorCodes(0)  // OK
	ErrorCodeNoGlobals                    = ErrorCodes(1)  // NoGlobals
	ErrorCodeGeoGlobals                   = ErrorCodes(2)  // GeoGlobals
	ErrorCodeGridGlobals                  = ErrorCodes(3)  // GridGlobals
	ErrorCodePOIGlobals                   = ErrorCodes(4)  // POIGlobals
	ErrorCodeInvalidID                    = ErrorCodes(5)  // InvalidID
	ErrorCodeNotImplemented               = ErrorCodes(6)  // NotImplemented
	ErrorCodeNoQuery                      = ErrorCodes(7)  // NoQuery
	ErrorCodeNoQueryAfterFormatting       = ErrorCodes(8)  // NoQueryAfterFormatting
	ErrorCodeNoAllowedInterps             = ErrorCodes(9)  // NoAllowedInterps
	ErrorCodeInvalidNumResultsRequested   = ErrorCodes(10) // InvalidNumResultsRequested
	ErrorCodeNoDataLoaded                 = ErrorCodes(11) // NoDataLoaded
	ErrorCodeDataLoad                     = ErrorCodes(12) // DataLoad
	ErrorCodeBadData                      = ErrorCodes(13) // BadData
	ErrorCodeFileMissing                  = ErrorCodes(14) // FileMissing
	ErrorCodeFolderIndexOOB               = ErrorCodes(15) // FolderIndexOOB
	ErrorCodeCityFileClientOOB            = ErrorCodes(16) // CityFileClientOOB
	ErrorCodeOOB                          = ErrorCodes(17) // OOB
	ErrorCodeFileIO                       = ErrorCodes(18) // FileIO
	ErrorCodeMemory                       = ErrorCodes(19) // Memory
	ErrorCodeNoPreviousSearch             = ErrorCodes(20) // NoPreviousSearch
	ErrorCodeThreadStart                  = ErrorCodes(21) // ThreadStart
	ErrorCodeThreadEnqueue                = ErrorCodes(22) // ThreadEnqueue
	ErrorCodeObjectNotInitialized         = ErrorCodes(23) // ObjectNotInitialized
	ErrorCodePOIData                      = ErrorCodes(24) // POIData
	ErrorCodeInternal                     = ErrorCodes(25) // Internal
	ErrorCodeQueryIsNotUTF8               = ErrorCodes(26) // QueryIsNotUTF8
	ErrorCodeInvalidQuery                 = ErrorCodes(27) // InvalidQuery
	ErrorCodeInvalidInterpRankingSettings = ErrorCodes(28) // InvalidInterpRankingSettings
	ErrorCodeInvalidParameters            = ErrorCodes(29) // InvalidParameters
	ErrorCodeSynonymVersionMismatch       = ErrorCodes(30) // SynonymVersionMismatch
	ErrorCodeSynonymAmbiguityMismatch     = ErrorCodes(31) // SynonymAmbiguityMismatch
	ErrorCodeFrequenciesDisabled          = ErrorCodes(32) // FrequenciesDisabled
	ErrorCodeTimeOut                      = ErrorCodes(33) // TimeOut
	ErrorCodeInvalidInputCountry          = ErrorCodes(34) // InvalidInputCountry
	ErrorCodeInvalidInputState            = ErrorCodes(35) // InvalidInputState
	ErrorCodeStateIsNotPartOfCountry      = ErrorCodes(36) // StateIsNotPartOfCountry
	ErrorCodeIndexVersionMismatch         = ErrorCodes(37) // IndexVersionMismatch
	ErrorCodeInvalidLatLonForRegion       = ErrorCodes(38) // InvalidLatLonForRegion
	ErrorCodeUnknown                      = ErrorCodes(39) // Unknown
)
