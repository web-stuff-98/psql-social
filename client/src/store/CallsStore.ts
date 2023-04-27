import { defineStore } from "pinia";
import {
  isCallAcknowledge,
  isCallResponse,
} from "../socketHandling/InterpretEvent";
import { useRouter } from "vue-router";
import useAuthStore from "./AuthStore";

type CallStoreState = { calls: { caller: string; called: string }[] };

const useCallStore = defineStore("calls", {
  state: () => ({ calls: [] } as CallStoreState),

  actions: {
    watchCalls(e: MessageEvent) {
      const router = useRouter();
      const authStore = useAuthStore();

      const msg = JSON.parse(e.data);
      if (!msg) return;
      if (isCallAcknowledge(msg)) {
        this.calls.push(msg.data);
      }
      if (isCallResponse(msg)) {
        const i = this.calls.findIndex(
          (c) => c.called === msg.data.called && c.caller === msg.data.caller
        );
        if (i !== -1) this.calls.splice(i, 1);
        if (msg.data.accept)
          router.push(
            `/call/${
              msg.data.called === authStore.uid
                ? msg.data.caller
                : msg.data.called
            }${msg.data.caller === authStore.uid ? "?initiator" : ""}`
          );
      }
    },
  },
});

export default useCallStore;
