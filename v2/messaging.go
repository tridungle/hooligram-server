package v2

import (
	"github.com/hooligram/hooligram-server/actions"
	"github.com/hooligram/hooligram-server/clients"
	"github.com/hooligram/hooligram-server/db"
	"github.com/hooligram/hooligram-server/delivery"
	"github.com/hooligram/hooligram-server/utils"
)

////////////////////////////////////////
// HANDLER: MESSAGING_DELIVER_SUCCESS //
////////////////////////////////////////

func handleMessagingDeliverSuccess(client *clients.Client, action *actions.Action) *actions.Action {
	if !client.IsSignedIn() {
		return nil
	}

	messageID, ok := action.Payload["message_id"].(float64)
	if !ok {
		return nil
	}

	recipientID := client.GetID()
	ok, err := db.UpdateReceiptDateDelivered(int(messageID), recipientID)
	if err != nil {
		utils.LogBody(v2Tag, "error updating receipt date delivered. "+err.Error())
		return nil
	}

	return nil
}

/////////////////////////////////////
// HANDLER: MESSAGING_SEND_REQUEST //
/////////////////////////////////////

func handleMessagingSendRequest(client *clients.Client, action *actions.Action) *actions.Action {
	actionID := action.ID
	if actionID == "" {
		return messagingSendFailure(client, actionID, "id not in action")
	}

	if !client.IsSignedIn() {
		return messagingSendFailure(client, actionID, "not signed in")
	}

	content, ok := action.Payload["content"].(string)
	if !ok {
		return messagingSendFailure(client, actionID, "content not in payload")
	}

	messageGroupID, ok := action.Payload["group_id"].(float64)
	if !ok {
		return messagingSendFailure(client, actionID, "group_id not in payload")
	}

	if !db.ReadIsClientInMessageGroup(int(client.GetID()), int(messageGroupID)) {
		return messagingSendFailure(client, actionID, "sender doesn't belong to message group")
	}

	message, err := db.CreateMessage(content, int(messageGroupID), client.GetID())
	if err != nil {
		utils.LogBody(v2Tag, "error creating message. "+err.Error())
		return messagingSendFailure(client, actionID, "server error")
	}

	messageGroupMemberIDs, err := db.ReadMessageGroupMemberIDs(message.MessageGroupID)
	if err != nil {
		utils.LogBody(v2Tag, "error finding meesage group member ids. "+err.Error())
		return messagingSendFailure(client, actionID, "server error")
	}

	for _, memberID := range messageGroupMemberIDs {
		db.CreateReceipt(message.ID, memberID)
	}

	delivery.GetMessageDeliveryChan() <- &delivery.MessageDelivery{
		Message:      message,
		RecipientIDs: messageGroupMemberIDs,
	}

	success := actions.MessagingSendSuccess(actionID, message.ID)
	client.WriteJSON(success)
	return success
}

////////////
// HELPER //
////////////

func messagingSendFailure(client *clients.Client, actionID, err string) *actions.Action {
	failure := actions.MessagingSendFailure(actionID, []string{err})
	client.WriteJSON(failure)
	return failure
}
