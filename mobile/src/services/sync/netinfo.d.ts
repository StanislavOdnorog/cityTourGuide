declare module '@react-native-community/netinfo' {
  export interface NetInfoState {
    isConnected: boolean | null;
    isInternetReachable: boolean | null;
    type: string;
  }

  type NetInfoChangeHandler = (state: NetInfoState) => void;

  interface NetInfo {
    addEventListener(listener: NetInfoChangeHandler): () => void;
    fetch(): Promise<NetInfoState>;
  }

  const netInfo: NetInfo;
  export default netInfo;
}
