#!/bin/bash

if [ "$#" -eq 0 ];then
   echo "Usage: $0 node.internal.dns.name"
   exit 1
fi


REGION=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/document|grep region|awk -F\" '{print $4}')
echo "Region is: ${REGION}"

NID=$(aws ec2  --region="$REGION" describe-instances --filters Name=private-dns-name,Values="$1" | jq -rM '.Reservations[].Instances[].InstanceId')

if [ -z $NID ]; then
    echo "Can not find node id. Exiting."
    exit 1
fi

echo "Node ID: ${NID}"

ASG=$(aws --region="$REGION" autoscaling describe-auto-scaling-instances | jq --arg NID "$NID" -rM '.AutoScalingInstances[]|select(.InstanceId==$NID).AutoScalingGroupName')

echo "ASG: ${ASG}"

TGARNS=()

shopt -s nocasematch
echo
while read TGARN; do
   #echo "Looking for $NID in TG: ${TGARN}"
   NHEALTH=$(aws --region=ap-south-1 elbv2  describe-target-health --target-group-arn="$TGARN" \
       | jq --arg NID "$NID" -rM '.TargetHealthDescriptions[]|select(.Target.Id==$NID).TargetHealth.State')
   if [[ $NHEALTH =~ healthy ]]; then
       echo "Found node ${1} in ${TGARN}"
       TGARNS+=("$TGARN")
   fi 
done < <(aws --region="$REGION" elbv2   describe-target-groups | jq -rM .TargetGroups[].TargetGroupArn)

#echo
#for TGARN in ${TGARNS[*]}; do
#    echo "Deregistering ${NID} from ${TGARN}"
#    aws --region="$REGION" elbv2 deregister-targets --target-group-arn="$TGARN" --targets Id="$NID"
#done

aws --region="$REGION" autoscaling detach-instances --instance-ids "$NID" --auto-scaling-group-name "$ASG" --no-should-decrement-desired-capacity

### Wait for node drain ###
TEMP_TGARNS=("${TGARNS[*]}")
while [ ${#TEMP_TGARNS[*]} -ne 0 ]; do
    TGARNS=("${TEMP_TGARNS[*]}")
    TEMP_TGARNS=()
    for TGARN in ${TGARNS[*]}; do
        NHEALTH=$(aws --region=ap-south-1 elbv2  describe-target-health --target-group-arn="$TGARN" \
            | jq --arg NID "$NID" -rM '.TargetHealthDescriptions[]|select(.Target.Id==$NID).TargetHealth.State') 
        if [[ $NHEALTH =~ healthy|drain ]]; then
            TEMP_TGARNS+=("$TGARN")
            echo "Waiting for insnance status in ${TGARN}: ${NHEALTH}"
        fi
    done
    echo -n "Sleep"
    for ii in {1..10}; do
        sleep .5
        echo -n "."
    done
    echo
done

echo
echo "Draining node ${NID} ${1}"
kubectl drain --ignore-daemonsets --delete-local-data --force "$1"
echo 

echo "Terminating instance ${NID} ${1}"
aws --region="$REGION" ec2 terminate-instances --instance-ids "$NID"

