package com.mushroomserchz.trae;

import com.mushroomserchz.trae.item.ModItems;
import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientTickEvents;
import net.minecraft.entity.EquipmentSlot;
import net.minecraft.entity.player.PlayerEntity;
import net.minecraft.item.Item;
import net.minecraft.particle.ParticleTypes;

public class TraeClientMod implements ClientModInitializer {
    @Override
    public void onInitializeClient() {
        ClientTickEvents.END_CLIENT_TICK.register(client -> {
            if (client.player == null || client.world == null) {
                return;
            }

            PlayerEntity player = client.player;
            if (isWearingFullSet(player) && client.world.getTime() % 2L == 0L) {
                double baseX = player.getX();
                double baseY = player.getBodyY(0.5D) + 0.25D;
                double baseZ = player.getZ();

                for (int i = 0; i < 18; i++) {
                    double angle = (Math.PI * 2D * i / 18D) + (client.world.getTime() * 0.08D);
                    double radius = 0.7D + ((i % 3) * 0.2D);
                    double x = baseX + Math.cos(angle) * radius;
                    double y = baseY + (Math.sin(client.world.getTime() * 0.2D + i) * 0.45D);
                    double z = baseZ + Math.sin(angle) * radius;

                    client.world.addParticle(ParticleTypes.END_ROD, x, y, z, 0.0D, 0.08D, 0.0D);
                    client.world.addParticle(ParticleTypes.ENCHANT, x, y + 0.35D, z, 0.0D, 0.06D, 0.0D);
                }
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
