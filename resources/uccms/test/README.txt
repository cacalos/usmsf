Usage for confctl :

- meta registration for example

  1) ./confctl add meta "SMSF" "Dicision Configureation" -F decisionconfig.json -V
  2) ./confctl add meta "SMSF" "Smsf Configuration" -F meta.json -V
  3) ./confctl add meta "SMSF" "Smsc Configuration" -F config.json -V

for 1)
	is added decision meta data that select sigtran or diameter for SMSF-SVC-POD

for 2)
	is added smsf meta data for SMSF Informations

for 3)
	is added SMSC meta data  for SMSC Informations

- configration registration for example

 1) ./confctl update config "SMSFDecision Configuration_keys_v1.0" -F decisionUpdate.json -V
 2) ./confctl update config "SMSFmsf Configuration_keys_v1.0" -F decisionUpdate.json -V
 3) ./confctl update config "Smsc Configuration_keys_v1.0" -F decisionUpdate.json -V

for 1)
	is added decision config data that select sigtran or diameter for SMSF-SVC-POD

for 2)
	is added smsf config data for SMSF Informations

for 3)
	is added SMSC config data  for SMSC Informations
