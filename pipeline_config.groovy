allow_scm_jenkinsfile = true
libraries{
  common
  make
  sonarqube{
    credential_id = "sonarqube"
    sonar_server = "SonarQube"
    enforce_quality_gate = false
  }
}
