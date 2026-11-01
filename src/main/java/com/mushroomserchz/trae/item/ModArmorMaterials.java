package com.mushroomserchz.trae.item;

import net.minecraft.entity.LivingEntity;
import net.minecraft.entity.effect.StatusEffectInstance;
import net.minecraft.entity.effect.StatusEffects;
import net.minecraft.entity.player.PlayerEntity;
import net.minecraft.item.ItemStack;
import net.minecraft.item.SwordItem;
import net.minecraft.item.ToolMaterial;
import net.minecraft.particle.ParticleTypes;
import net.minecraft.server.world.ServerWorld;
import net.minecraft.sound.SoundCategory;
import net.minecraft.sound.SoundEvents;
import net.minecraft.util.math.Box;
import net.minecraft.util.math.Vec3d;

import java.util.List;

public class GodslayerSwordItem extends SwordItem {
    public GodslayerSwordItem(ToolMaterial toolMaterial, int attackDamage, float attackSpeed, Settings settings) {
        super(toolMaterial, attackDamage, attackSpeed, settings);
    }

    @Override
    public boolean postHit(ItemStack stack, LivingEntity target, LivingEntity attacker) {
        if (!(attacker.getWorld() instanceof ServerWorld serverWorld)) {
            return super.postHit(stack, target, attacker);
        }

        Vec3d center = target.getPos();
        spawnEclipseBurst(serverWorld, center, 8.0D, 80);
        target.damage(target.getDamageSources().magic(), 99999.0F);
        target.addStatusEffect(new StatusEffectInstance(StatusEffects.WITHER, 100, 6, false, false));
        target.addStatusEffect(new StatusEffectInstance(StatusEffects.SLOWNESS, 100, 9, false, false));
        target.addStatusEffect(new StatusEffectInstance(StatusEffects.BLINDNESS, 60, 0, false, true));

        Box area = target.getBoundingBox().expand(12.0D);
        List<LivingEntity> nearby = serverWorld.getEntitiesByClass(
                LivingEntity.class,
                area,
                entity -> entity != attacker && entity.isAlive()
        );

        for (LivingEntity nearbyEntity : nearby) {
            nearbyEntity.damage(nearbyEntity.getDamageSources().magic(), 99999.0F);
            nearbyEntity.addStatusEffect(new StatusEffectInstance(StatusEffects.WITHER, 100, 8, false, false));
            nearbyEntity.addStatusEffect(new StatusEffectInstance(StatusEffects.SLOWNESS, 60, 9, false, false));
            spawnEclipseBurst(serverWorld, nearbyEntity.getPos(), 6.0D, 28);
        }

        if (attacker instanceof PlayerEntity player) {
            player.heal(30.0F);
            player.getWorld().playSound(
                    null,
                    target.getBlockPos(),
                    SoundEvents.ENTITY_LIGHTNING_BOLT_IMPACT,
                    SoundCategory.PLAYERS,
                    1.5F,
                    1.0F
            );
        }

        return true;
    }

    private static void spawnEclipseBurst(ServerWorld world, Vec3d center, double radius, int particleCount) {
        for (int i = 0; i < particleCount; i++) {
            double angle = (Math.PI * 2D * i / particleCount) + (world.getTime() * 0.08D);
            double offsetX = Math.cos(angle) * radius;
            double offsetZ = Math.sin(angle) * radius;
            double y = center.y + (i % 3) * 0.2D;

            world.spawnParticles(ParticleTypes.SOUL_FIRE_FLAME, center.x + offsetX, y, center.z + offsetZ, 1, 0.0D, 0.1D, 0.0D, 0.0D);
            world.spawnParticles(ParticleTypes.END_ROD, center.x + offsetX * 0.7D, y + 0.35D, center.z + offsetZ * 0.7D, 1, 0.0D, 0.0D, 0.0D, 0.0D);
            world.spawnParticles(ParticleTypes.ENCHANT, center.x + offsetX * 0.5D, y + 0.65D, center.z + offsetZ * 0.5D, 1, 0.0D, 0.0D, 0.0D, 0.0D);
        }
    }
}
