import { reactive } from "vue";

interface IUserDropdownStore {
  open: boolean;
  subject: string;
  roomId: string;
  openOnSubject: (uid: string, roomId?: string) => void;
}

export const userdropdownStore: IUserDropdownStore = reactive({
  open: false,
  subject: "",
  roomId: "",
  openOnSubject: (uid: string, roomId?: string) => {
    userdropdownStore.roomId = roomId || "";
    userdropdownStore.subject = uid;
    userdropdownStore.open = true;
  },
});
