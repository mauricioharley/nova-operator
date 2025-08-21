#!/bin/bash
set -ex

# Use kubectl if oc is not available
if command -v oc &> /dev/null; then
    CLI_CMD="oc"
elif command -v kubectl &> /dev/null; then
    CLI_CMD="kubectl"
else
    echo "Neither oc nor kubectl found, skipping webhook cleanup"
    exit 0
fi

$CLI_CMD delete validatingwebhookconfiguration/vnova.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnova.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovaapi.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovaapi.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovacell.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovacell.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovaconductor.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovaconductor.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovametadata.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovametadata.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovanovncproxy.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovanovncproxy.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovascheduler.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovascheduler.kb.io --ignore-not-found
$CLI_CMD delete validatingwebhookconfiguration/vnovacompute.kb.io --ignore-not-found
$CLI_CMD delete mutatingwebhookconfiguration/mnovacompute.kb.io --ignore-not-found
