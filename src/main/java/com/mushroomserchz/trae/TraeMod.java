package com.mushroomserchz.trae;

import com.mushroomserchz.trae.item.ModItems;
import net.fabricmc.api.ModInitializer;
import net.fabricmc.fabric.api.itemgroup.v1.FabricItemGroup;
import net.minecraft.item.ItemGroup;
import net.minecraft.item.ItemStack;
import net.minecraft.registry.Registries;
import net.minecraft.registry.Registry;
import net.minecraft.util.Identifier;

public class TraeMod implements ModInitializer {
    public static final String MOD_ID = "trae";

    public static final ItemGroup TRAE_GROUP = FabricItemGroup.builder(id("creative_tab"))
            .icon(() -> new ItemStack(ModItems.GODS_EDGE))
            .entries((displayContext, entries) -> {
                entries.add(ModItems.GODS_EDGE);
                entries.add(ModItems.ABYSSAL_HELMET);
                entries.add(ModItems.ABYSSAL_CHESTPLATE);
                entries.add(ModItems.ABYSSAL_LEGGINGS);
                entries.add(ModItems.ABYSSAL_BOOTS);
            })
            .build();

    @Override
    public void onInitialize() {
        ModItems.register();
        Registry.register(Registries.ITEM_GROUP, id("creative_tab"), TRAE_GROUP);
    }

    public static Identifier id(String path) {
        return Identifier.of(MOD_ID, path);
    }
}
