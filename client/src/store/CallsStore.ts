import { reactive } from "vue";

export const pendingCallsStore: {
  caller: string;
  called: string;
}[] = reactive([]);