package com.mushroomserchz.trae;

import com.mushroomserchz.trae.item.ModItems;
import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientTickEvents;
import net.minecraft.entity.EquipmentSlot;
import net.minecraft.entity.player.PlayerEntity;
import net.minecraft.item.Item;
import net.minecraft.item.ItemStack;
import net.minecraft.particle.ParticleTypes;

public class TraeClientMod implements ClientModInitializer {
    @Override
    public void onInitializeClient() {
        ClientTickEvents.END_CLIENT_TICK.register(client -> {
            if (client.player == null || client.world == null) {
                return;
            }

            if (isWearingFullSet(client.player) && client.world.getTime() % 3L == 0L) {
                double x = client.player.getX() + (client.world.random.nextDouble() - 0.5D) * 0.8D;
                double y = client.player.getBodyY(0.5D) + 0.5D;
                double z = client.player.getZ() + (client.world.random.nextDouble() - 0.5D) * 0.8D;

                client.world.addParticle(ParticleTypes.END_ROD, x, y, z, 0.0D, 0.05D, 0.0D);
                client.world.addParticle(ParticleTypes.ENCHANT, x, y + 0.3D, z, 0.0D, 0.08D, 0.0D);
            }
        });
    }

    private static boolean isWearingFullSet(PlayerEntity player) {
        return isArmorPiece(player, EquipmentSlot.HEAD, ModItems.ABYSSAL_HELMET)
                && isArmorPiece(player, EquipmentSlot.CHEST, ModItems.ABYSSAL_CHESTPLATE)
                && isArmorPiece(player, EquipmentSlot.LEGS, ModItems.ABYSSAL_LEGGINGS)
                && isArmorPiece(player, EquipmentSlot.FEET, ModItems.ABYSSAL_BOOTS);
    }

    private static boolean isArmorPiece(PlayerEntity player, EquipmentSlot slot, Item item) {
        return player.getEquippedStack(slot).isOf(item);
    }
}
