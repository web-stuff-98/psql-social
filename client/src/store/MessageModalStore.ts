import { reactive } from "vue";
import { IResMsg } from "../interfaces/GeneralInterfaces";

type MessageModalStore = {
  confirmationCallback: Function;
  cancellationCallback?: Function;
  show: boolean;
  msg: IResMsg;
};

const messageModalStore = reactive<MessageModalStore>({
  confirmationCallback: () => {},
  cancellationCallback: undefined,
  show: false,
  msg: {},
});

export default messageModalStore;
