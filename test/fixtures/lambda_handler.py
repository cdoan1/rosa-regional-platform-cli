import json

def handler(event, context):
    """
    Simple Lambda handler for e2e testing.
    Returns the event payload along with a success message.
    """
    return {
        'statusCode': 200,
        'body': json.dumps({
            'message': 'success',
            'event': event
        })
    }
