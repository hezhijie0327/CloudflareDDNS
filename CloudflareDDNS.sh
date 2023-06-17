#!/bin/bash

# Current Version: 1.2.6

## How to get and use?
# git clone "https://github.com/hezhijie0327/CloudflareDDNS.git" && bash ./CloudflareDDNS/CloudflareDDNS.sh -e demo@zhijie.online -k 123defghijk4567pqrstuvw890 -z zhijie.online -r demo.zhijie.online -t A -l 3600 -i "" -p false -m update

## Parameter
while getopts e:i:k:l:m:p:r:t:z: GetParameter; do
    case ${GetParameter} in
        e) XAuthEmail="${OPTARG}";;
        i) StaticIP="${OPTARG}";;
        k) XAuthKey="${OPTARG}";;
        l) TTL="${OPTARG}";;
        m) RunningMode="${OPTARG}";;
        p) ProxyStatus="${OPTARG}";;
        r) RecordName="${OPTARG}";;
        t) Type="${OPTARG}";;
        z) ZoneName="${OPTARG}";;
    esac
done

## Function
# Check Configuration Validity
function CheckConfigurationValidity() {
    if [ "${XAuthEmail}" == "" ]; then
        echo "An error occurred during processing. Missing (XAuthEmail) value, please check it and try again."
        exit 1
    fi
    if [ "${XAuthKey}" == "" ]; then
        echo "An error occurred during processing. Missing (XAuthKey) value, please check it and try again."
        exit 1
    fi
    if [ "${ZoneName}" == "" ]; then
        echo "An error occurred during processing. Missing (ZoneName) value, please check it and try again."
        exit 1
    fi
    if [ "${RecordName}" == "" ]; then
        echo "An error occurred during processing. Missing (RecordName) value, please check it and try again."
        exit 1
    fi
    if [ "${RunningMode}" == "" ]; then
        echo "An error occurred during processing. Missing (RunningMode) value, please check it and try again."
        exit 1
    elif [ "${RunningMode}" != "create" ] && [ "${RunningMode}" != "update" ] && [ "${RunningMode}" != "delete" ]; then
        echo "An error occurred during processing. Invalid (RunningMode) value, please check it and try again."
        exit 1
    fi
    if [ "${RunningMode}" == "create" ] || [ "${RunningMode}" == "update" ]; then
        if [ "${Type}" == "" ]; then
            echo "An error occurred during processing. Missing (Type) value, please check it and try again."
            exit 1
        elif [ "${Type}" != "A" ] && [ "${Type}" != "AAAA" ] && [ "${Type}" != "A_AAAA" ]; then
            echo "An error occurred during processing. Invalid (Type) value, please check it and try again."
            exit 1
        fi
        if [ "${TTL}" == "" ]; then
            echo "An error occurred during processing. Missing (TTL) value, please check it and try again."
            exit 1
        elif [ "${TTL}" != "1" ] && [ "${TTL}" != "120" ] && [ "${TTL}" != "300" ] && [ "${TTL}" != "600" ] && [ "${TTL}" != "900" ] && [ "${TTL}" != "1800" ] && [ "${TTL}" != "3600" ] && [ "${TTL}" != "7200" ] && [ "${TTL}" != "18000" ] && [ "${TTL}" != "43200" ] && [ "${TTL}" != "86400" ]; then
            echo "An error occurred during processing. Invalid (TTL) value, please check it and try again."
            exit 1
        fi
        if [ "${ProxyStatus}" == "" ]; then
            echo "An error occurred during processing. Missing (ProxyStatus) value, please check it and try again."
            exit 1
        elif [ "${ProxyStatus}" != "true" ] && [ "${ProxyStatus}" != "false" ]; then
            echo "An error occurred during processing. Invalid (ProxyStatus) value, please check it and try again."
            exit 1
        fi
    fi
}
# Check Environment Validity
function CheckEnvironmentValidity() {
    which "curl" > "/dev/null" 2>&1
    if [ "$?" -eq "1" ]; then
        echo "curl has not been installed!"
        exit 1
    fi
    which "dig" > "/dev/null" 2>&1
    if [ "$?" -eq "1" ]; then
        echo "dig has not been installed!"
        exit 1
    fi
    which "jq" > "/dev/null" 2>&1
    if [ "$?" -eq "1" ]; then
        echo "jq has not been installed!"
        exit 1
    fi
}
# Get Account Name
function GetAccountName() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X GET "https://api.cloudflare.com/client/v4/accounts?page=1&per_page=5&direction=desc" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | {name} | .name')" == "" ]; then
            echo "false"
        else
            echo "$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | {name} | .name')"
        fi
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
# Get Zone ID
function GetZoneID() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X GET "https://api.cloudflare.com/client/v4/zones?name=${ZoneName}" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | {id} | .id')" == "" ]; then
            echo "false"
        else
            echo "$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | {id} | .id')"
        fi
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
function GetRecordID() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X GET "https://api.cloudflare.com/client/v4/zones/${ZoneID}/dns_records?name=${RecordName}" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        if [ "${Type}" == "A" ]; then
            CloudflareAPIv4Response=$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | select(.type == "A") | {id} | .id')
        else
            CloudflareAPIv4Response=$(echo ${CloudflareAPIv4Response} | jq -r '.result[] | select(.type == "AAAA") | {id} | .id')
        fi
        if [ "${CloudflareAPIv4Response}" == "" ]; then
            echo "false"
        else
            echo "${CloudflareAPIv4Response}"
        fi
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
# Get DNS Record
function GetDNSRecord() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X GET "https://api.cloudflare.com/client/v4/zones/${ZoneID}/dns_records/${RecordID}" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.result.content')" == "" ]; then
            echo "false"
        else
            echo "$(echo ${CloudflareAPIv4Response} | jq -r '.result.content')"
        fi
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
# Get WAN IP
function GetWANIP() {
    function CheckIPValid() {
        if [ "${Type}" == "A" ]; then
            echo "${IP_RESULT}" | grep -E "^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$"
        else
            echo "${IP_RESULT}" | grep -E "^(([0-9a-f]{1,4}:){7,7}[0-9a-f]{1,4}|([0-9a-f]{1,4}:){1,7}:|([0-9a-f]{1,4}:){1,6}:[0-9a-f]{1,4}|([0-9a-f]{1,4}:){1,5}(:[0-9a-f]{1,4}){1,2}|([0-9a-f]{1,4}:){1,4}(:[0-9a-f]{1,4}){1,3}|([0-9a-f]{1,4}:){1,3}(:[0-9a-f]{1,4}){1,4}|([0-9a-f]{1,4}:){1,2}(:[0-9a-f]{1,4}){1,5}|[0-9a-f]{1,4}:((:[0-9a-f]{1,4}){1,6})|:((:[0-9a-f]{1,4}){1,7}|:)|fe80:(:[0-9a-f]{0,4}){0,4}%[0-9a-z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-f]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$"
        fi
    }
    if [ "${Type}" == "A" ]; then
        IPv4_v6="4"
    else
        IPv4_v6="6"
    fi
    if [ "${StaticIP}" == "" ]; then
        IP_RESULT=$(dig -${IPv4_v6:-4} +short TXT @ns1.google.com o-o.myaddr.l.google.com | tr -d '"')
        if [[ $(CheckIPValid) = "" ]]; then
            IP_RESULT=$(dig -${IPv4_v6:-4} +short ANY @resolver1.opendns.com myip.opendns.com)
            if [[ $(CheckIPValid) = "" ]]; then
                IP_RESULT=$(curl -${IPv4_v6:-4} -s --connect-timeout 15 "https://api64.ipify.org")
                if [[ $(CheckIPValid) = "" ]]; then
                    echo "invalid"
                else
                    echo "${IP_RESULT}"
                fi
            else
                echo "${IP_RESULT}"
            fi
        else
            echo "${IP_RESULT}"
        fi
    else
        if [ $(echo "${StaticIP}" | grep ",") != "" ]; then
            if [ "${Type}" == "A" ]; then
                IP_RESULT=$(echo "${StaticIP}" | cut -d ',' -f 1)
            else
                IP_RESULT=$(echo "${StaticIP}" | cut -d ',' -f 2)
            fi
            if [[ $(CheckIPValid) = "" ]]; then
                echo "invalid"
            else
                echo "${IP_RESULT}"
            fi
        else
            IP_RESULT="${StaticIP}"
            if [[ $(CheckIPValid) = "" ]]; then
                echo "invalid"
            else
                echo "${IP_RESULT}"
            fi
        fi
    fi
}
# Get POST Response
function GetPOSTResponse() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X POST "https://api.cloudflare.com/client/v4/zones/${ZoneID}/dns_records" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json" --data "{\"type\":\"${Type}\",\"name\":\"${RecordName}\",\"content\":\"${WANIP}\",\"ttl\":${TTL},\"proxied\":${ProxyStatus}}")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        echo "true"
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
# Get PUT Response
function GetPUTResponse() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X PUT "https://api.cloudflare.com/client/v4/zones/${ZoneID}/dns_records/${RecordID}" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json" --data "{\"type\":\"${Type}\",\"name\":\"${RecordName}\",\"content\":\"${WANIP}\",\"ttl\":${TTL},\"proxied\":${ProxyStatus}}")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        echo "true"
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}
# Get DELETE Response
function GetDELETEResponse() {
    CloudflareAPIv4Response=$(curl -s --connect-timeout 15 -X DELETE "https://api.cloudflare.com/client/v4/zones/${ZoneID}/dns_records/${RecordID}" -H "X-Auth-Email: ${XAuthEmail}" -H "X-Auth-Key: ${XAuthKey}" -H "Content-Type: application/json")
    if [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "true" ]; then
        echo "true"
    elif [ "$(echo ${CloudflareAPIv4Response} | jq -r '.success')" == "false" ]; then
        echo "false"
    else
        echo "invalid"
    fi
}

## Process
# Call CheckConfigurationValidity
CheckConfigurationValidity
# Call CheckEnvironmentValidity
CheckEnvironmentValidity
if [ "${RunningMode}" == "create" ]; then
    # Call GetAccountName
    AccountName=$(GetAccountName)
    if [ "${AccountName}" == "invalid" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check your network connectivity, and try again."
        exit 1
    elif [ "${AccountName}" == "false" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check (XAuthEmail) and (XAuthKey) value, and try again."
        exit 1
    else
        echo "Current Account Name: ${AccountName}"
        # Call GetZoneID
        ZoneID=$(GetZoneID)
        if [ "${ZoneID}" == "invalid" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check your network connectivity, and try again."
            exit 1
        elif [ "${ZoneID}" == "false" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check (ZoneName) value, and try again."
            exit 1
        else
            function CreateQueue() {
                # Call GetRecordID
                RecordID=$(GetRecordID)
                if [ "${RecordID}" == "invalid" ]; then
                    echo "An error occurred during processing. Invalid (RecordID) value, please check your network connectivity, and try again."
                    exit 1
                elif [ "${RecordID}" != "invalid" ] && [ "${RecordID}" != "false" ]; then
                    echo "An error occurred during processing. ${RecordName} has been existed."
                    exit 1
                else
                    # Call GetWANIP
                    WANIP=$(GetWANIP)
                    if [ "${WANIP}" == "invalid" ]; then
                        if [ "${Type}" == "A" ]; then
                            echo "An error occurred during processing. Invalid (WANIP) value, please check your IPv4 connectivity."
                            exit 1
                        else
                            echo "An error occurred during processing. Invalid (WANIP) value, please check your IPv6 connectivity."
                            exit 1
                        fi
                    else
                        echo "Current WAN IP: ${WANIP}"
                        # Call GetPOSTResponse
                        POSTResponse=$(GetPOSTResponse)
                        if [ "${POSTResponse}" == "true" ]; then
                            echo "No error occurred during processing. ${RecordName} has been created."
                        else
                            echo "An error occurred during processing. Invalid (POSTResponse) value, please check your network connectivity, and try again."
                            exit 1
                        fi
                    fi
                fi
            }
            echo "Current Zone ID: ${ZoneID}"
            if [ "${Type}" == "A_AAAA" ]; then
                Type="A" && CreateQueue
                Type="AAAA" && CreateQueue
            else
                CreateQueue
            fi
        fi
    fi
elif [ "${RunningMode}" == "update" ]; then
    # Call GetAccountName
    AccountName=$(GetAccountName)
    if [ "${AccountName}" == "invalid" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check your network connectivity, and try again."
        exit 1
    elif [ "${AccountName}" == "false" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check (XAuthEmail) and (XAuthKey) value, and try again."
        exit 1
    else
        echo "Current Account Name: ${AccountName}"
        # Call GetZoneID
        ZoneID=$(GetZoneID)
        if [ "${ZoneID}" == "invalid" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check your network connectivity, and try again."
            exit 1
        elif [ "${ZoneID}" == "false" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check (ZoneName) value, and try again."
            exit 1
        else
            function UpdateQueue() {
                # Call GetRecordID
                RecordID=$(GetRecordID)
                if [ "${RecordID}" == "invalid" ]; then
                    echo "An error occurred during processing. Invalid (RecordID) value, please check your network connectivity, and try again."
                    exit 1
                elif [ "${RecordID}" == "false" ]; then
                    echo "An error occurred during processing. ${RecordName} has not been existed."
                    exit 1
                else
                    echo "Current Record ID: ${RecordID}"
                    # Call GetWANIP
                    WANIP=$(GetWANIP)
                    if [ "${WANIP}" == "invalid" ]; then
                        if [ "${Type}" == "A" ]; then
                            echo "An error occurred during processing. Invalid (WANIP) value, please check your IPv4 connectivity."
                            exit 1
                        else
                            echo "An error occurred during processing. Invalid (WANIP) value, please check your IPv6 connectivity."
                            exit 1
                        fi
                    else
                        echo "Current WAN IP: ${WANIP}"
                        # Call GetDNSRecord
                        DNSRecord=$(GetDNSRecord)
                        if [ "${DNSRecord}" == "invalid" ]; then
                            echo "An error occurred during processing. Invalid (DNSRecord) value, please check your network connectivity, and try again."
                            exit 1
                        elif [ "${DNSRecord}" == "false" ]; then
                            echo "An error occurred during processing. Invalid (DNSRecord) value, please check (ZoneName) and (RecordName) value, and try again."
                            exit 1
                        else
                            if [ "${DNSRecord}" == "${WANIP}" ]; then
                                echo "No error occurred during processing. WAN IP has not been changed."
                            else
                                echo "Current DNS Record: ${DNSRecord}"
                                # Call GetPOSTResponse
                                PUTResponse=$(GetPUTResponse)
                                if [ "${PUTResponse}" == "true" ]; then
                                    echo "No error occurred during processing. ${RecordName} has been updated."
                                else
                                    echo "An error occurred during processing. Invalid (PUTResponse) value, please check your network connectivity, and try again."
                                    exit 1
                                fi
                            fi
                        fi
                    fi
                fi
            }
            echo "Current Zone ID: ${ZoneID}"
            if [ "${Type}" == "A_AAAA" ]; then
                Type="A" && UpdateQueue
                Type="AAAA" && UpdateQueue
            else
                UpdateQueue
            fi
        fi
    fi
else
    # Call GetAccountName
    AccountName=$(GetAccountName)
    if [ "${AccountName}" == "invalid" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check your network connectivity, and try again."
        exit 1
    elif [ "${AccountName}" == "false" ]; then
        echo "An error occurred during processing. Invalid (AccountName) value, please check (XAuthEmail) and (XAuthKey) value, and try again."
        exit 1
    else
        echo "Current Account Name: ${AccountName}"
        # Call GetZoneID
        ZoneID=$(GetZoneID)
        if [ "${ZoneID}" == "invalid" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check your network connectivity, and try again."
            exit 1
        elif [ "${ZoneID}" == "false" ]; then
            echo "An error occurred during processing. Invalid (ZoneID) value, please check (ZoneName) value, and try again."
            exit 1
        else
            function DeleteQueue() {
                # Call GetRecordID
                RecordID=$(GetRecordID)
                if [ "${RecordID}" == "invalid" ]; then
                    echo "An error occurred during processing. Invalid (RecordID) value, please check your network connectivity, and try again."
                    exit 1
                elif [ "${RecordID}" == "false" ]; then
                    echo "An error occurred during processing. ${RecordName} has not been existed."
                    exit 1
                else
                    echo "Current Record ID: ${RecordID}"
                    # Call GetDELETEResponse
                    DELETEResponse=$(GetDELETEResponse)
                    if [ "${DELETEResponse}" == "true" ]; then
                        echo "No error occurred during processing. ${RecordName} has been deleted."
                    else
                        echo "An error occurred during processing. Invalid (DELETEResponse) value, please check your network connectivity, and try again."
                        exit 1
                    fi
                fi
            }
            echo "Current Zone ID: ${ZoneID}"
            if [ "${Type}" == "A_AAAA" ]; then
                Type="A" && DeleteQueue
                Type="AAAA" && DeleteQueue
            else
                DeleteQueue
            fi
        fi
    fi
fi
