import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  Pressable,
  ActivityIndicator,
  Alert,
  Platform,
} from 'react-native';
import { verifyPurchase, fetchPurchaseStatus } from '@/api';
import { usePurchaseStore, PRODUCT_CATALOG, type ProductInfo } from '@/store/usePurchaseStore';

interface PaywallProps {
  visible: boolean;
  onClose: () => void;
  cityId?: number;
}

export function Paywall({ visible, onClose, cityId }: PaywallProps) {
  const [purchasing, setPurchasing] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<ProductInfo | null>(null);
  const setStatus = usePurchaseStore((s) => s.setStatus);
  const status = usePurchaseStore((s) => s.status);

  const freeLeft = status?.free_stories_left ?? 0;

  const handlePurchase = async (product: ProductInfo) => {
    setSelectedProduct(product);
    setPurchasing(true);

    try {
      // In production, this would use react-native-iap to get a receipt
      // from the native store. For now, we simulate the flow.
      const platform = Platform.OS === 'ios' ? 'ios' : 'android';

      // Step 1: Initiate IAP purchase via react-native-iap
      // const purchase = await requestPurchase({ sku: product.id });
      // Step 2: Send receipt to backend for verification
      const transactionId = `${platform}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
      const receipt = `receipt_${transactionId}`;

      await verifyPurchase({
        platform: platform as 'ios' | 'android',
        transaction_id: transactionId,
        receipt,
        type: product.type,
        city_id: product.type === 'city_pack' ? cityId : undefined,
        price: parseFloat(product.price.replace(/[^0-9.]/g, '')),
      });

      // Step 3: Refresh purchase status
      const updatedStatus = await fetchPurchaseStatus();
      setStatus(updatedStatus);

      Alert.alert('Purchase Successful', `You now have ${product.title}!`);
      onClose();
    } catch {
      Alert.alert('Purchase Failed', 'Something went wrong. Please try again.');
    } finally {
      setPurchasing(false);
      setSelectedProduct(null);
    }
  };

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <View style={styles.overlay}>
        <View style={styles.container}>
          <Pressable style={styles.closeButton} onPress={onClose}>
            <Text style={styles.closeText}>X</Text>
          </Pressable>

          <Text style={styles.title}>Unlock More Stories</Text>
          <Text style={styles.subtitle}>
            {freeLeft === 0
              ? "You've used all your free stories for today."
              : `${freeLeft} free stories remaining today.`}
          </Text>
          <Text style={styles.description}>
            Upgrade to enjoy unlimited stories about the places you explore.
          </Text>

          <View style={styles.products}>
            {PRODUCT_CATALOG.map((product) => {
              const isSelected = selectedProduct?.id === product.id;
              const isDisabled = purchasing && !isSelected;

              return (
                <Pressable
                  key={product.id}
                  style={[styles.productCard, isDisabled && styles.productDisabled]}
                  onPress={() => void handlePurchase(product)}
                  disabled={purchasing}
                >
                  {purchasing && isSelected ? (
                    <ActivityIndicator color="#FFFFFF" />
                  ) : (
                    <>
                      <Text style={styles.productTitle}>{product.title}</Text>
                      <Text style={styles.productDescription}>{product.description}</Text>
                      <Text style={styles.productPrice}>{product.price}</Text>
                    </>
                  )}
                </Pressable>
              );
            })}
          </View>

          <Pressable style={styles.restoreButton} onPress={onClose}>
            <Text style={styles.restoreText}>Restore Purchases</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.7)',
    justifyContent: 'flex-end',
  },
  container: {
    backgroundColor: '#1A1A1A',
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    paddingHorizontal: 24,
    paddingTop: 20,
    paddingBottom: 40,
  },
  closeButton: {
    position: 'absolute',
    top: 16,
    right: 16,
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: '#333333',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1,
  },
  closeText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '600',
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#FFFFFF',
    textAlign: 'center',
    marginTop: 8,
  },
  subtitle: {
    fontSize: 15,
    color: '#FF9500',
    textAlign: 'center',
    marginTop: 8,
  },
  description: {
    fontSize: 14,
    color: '#AAAAAA',
    textAlign: 'center',
    marginTop: 4,
    marginBottom: 20,
  },
  products: {
    gap: 12,
  },
  productCard: {
    backgroundColor: '#2A2A2A',
    borderRadius: 16,
    padding: 16,
    minHeight: 80,
    justifyContent: 'center',
  },
  productDisabled: {
    opacity: 0.5,
  },
  productTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: '#FFFFFF',
  },
  productDescription: {
    fontSize: 13,
    color: '#AAAAAA',
    marginTop: 2,
  },
  productPrice: {
    fontSize: 17,
    fontWeight: '700',
    color: '#4CAF50',
    position: 'absolute',
    right: 16,
    top: 16,
  },
  restoreButton: {
    marginTop: 16,
    alignItems: 'center',
    paddingVertical: 8,
  },
  restoreText: {
    fontSize: 14,
    color: '#666666',
    textDecorationLine: 'underline',
  },
});
