package org.example.client.bot;

public class PendingAction {
    private final PendingActionType type;
    private final String data;

    public PendingAction(PendingActionType type, String data) {
        this.type = type;
        this.data = data;
    }

    public PendingActionType type() {
        return type;
    }

    public String data() {
        return data;
    }
}
